package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	version            = "v0.1.0"
	cacheDirRoot       = "/tmp"
	cacheFileInstances = "instances.json"
)

type Instance struct {
	InstanceId string            `json:"instance_id"`
	Tags       map[string]string `json:"tags"`
	Name       string            `json:"name"`
	Roles      []string          `json:"roles"`
	VpcId      string            `json:"vpc_id"`
	Vpc        *Vpc              `json:"vpc"`
}

func (i *Instance) Fqdn(config *Config) string {
	tmpl, err := template.New("fqdn").Parse(config.TemplateFqdn)
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, i)
	if err != nil {
		log.Fatal(err)
	}
	return buf.String()
}

type Vpc struct {
	VpcId     string            `json:"vpc_id"`
	Tags      map[string]string `json:"tags"`
	Name      string            `json:"name"`
	ShortName string            `json:"short_name"`
}

type ReadOutput struct {
	IsCached  bool
	Instances map[string]*Instance
}

func (ro *ReadOutput) BuildCandidates() string {
	var candidates []string
	for instanceId, instance := range ro.Instances {
		line := fmt.Sprintf("%.30s %.10s %s", instance.Name, instance.Vpc.Name, instanceId)
		candidates = append(candidates, line)
	}
	sort.Strings(candidates)
	return strings.Join(candidates, "\n")
}

type Config struct {
	PurgeCache   bool
	Region       string
	SshBin       string
	TemplateFqdn string
}

func NewConfig() *Config {
	config := &Config{}
	flag.StringVar(&config.TemplateFqdn, "template-fqdn", "{{.Name}}.aws.example.com",
		"a template for building FQDN for servers based on text/template interface")
	flag.StringVar(&config.Region, "region", "us-east-1", "AWS Region for the session")
	flag.StringVar(&config.SshBin, "ssh-bin", "ssh", "a path to the binary for SSH")
	flag.BoolVar(&config.PurgeCache, "purge-cache", false, "purge local cache of AWS API calls")
	flag.Parse()
	return config
}

func Merge(instances map[string]*Instance, vpcs map[string]*Vpc) map[string]*Instance {
	for _, instance := range instances {
		if vpc, ok := vpcs[instance.VpcId]; ok {
			instance.Vpc = vpc
		} else {
			log.Fatalf("Vpc not found for instance_id=%s, vpc_id=%s\n",
				instance.InstanceId, instance.VpcId)
		}
	}
	return instances
}

func IsCacheEntry(entry fs.FileInfo, prefix string) bool {
	return entry.IsDir() && strings.HasPrefix(entry.Name(), prefix)
}

func IsExpired(entry fs.FileInfo) bool {
	expiresIn := time.Duration(24*60*60) * time.Second // 1day
	expiresTime := entry.ModTime().Add(expiresIn)
	return expiresTime.Before(time.Now())
}

func main() {
	config := NewConfig()

	readOutput := readInstances(config)
	candidates := readOutput.BuildCandidates()

	pecoResult := runPeco(candidates)

	fqdns := buildFqdns(config, readOutput, pecoResult)
	servers := strings.Join(fqdns, " ")

	runSSH(config, servers)
}

func readInstances(config *Config) *ReadOutput {
	prefix := fmt.Sprintf("%s-%s-%s", "go-awssh", version, config.Region)

	if config.PurgeCache {
		os.RemoveAll(cacheDirRoot)
	} else if cached := readFromCache(prefix); cached != nil {
		return cached
	}

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("%s-%s", prefix, "*"))
	if err != nil {
		log.Fatal(err)
	}
	instances := describeInstances(config)
	writeCache(tmpDir, cacheFileInstances, instances)
	return &ReadOutput{IsCached: false, Instances: instances}
}

func runPeco(candidates string) string {
	pecoBin := "peco"
	_, err := exec.LookPath(pecoBin)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(pecoBin)
	cmd.Stdin = strings.NewReader(candidates)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return out.String()
}

func runSSH(config *Config, servers string) {
	_, err := exec.LookPath(config.SshBin)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(config.SshBin, servers)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func buildFqdns(config *Config, readOutput *ReadOutput, pecoResult string) []string {
	var fqdns []string
	selectedInstances := strings.Split(pecoResult, "\n")
	for _, selection := range selectedInstances {
		if len(selection) == 0 {
			continue
		}
		fields := strings.Split(selection, " ")
		instanceId := fields[2]
		instance := readOutput.Instances[instanceId]
		if instance == nil {
			log.Fatalf("instance (instance_id=%s) not found\n", instanceId)
		}
		fqdns = append(fqdns, instance.Fqdn(config))
	}
	return fqdns
}

func readFromCache(prefix string) *ReadOutput {
	entries, err := ioutil.ReadDir(cacheDirRoot)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		if IsCacheEntry(entry, prefix) && !IsExpired(entry) {
			cacheInstancesPath := path.Join(cacheDirRoot, entry.Name(), cacheFileInstances)
			cacheInstances, err := os.ReadFile(cacheInstancesPath)
			if err != nil {
				log.Fatal(err)
			}
			var instances map[string]*Instance
			err = json.Unmarshal(cacheInstances, &instances)
			if err != nil {
				log.Fatal(err)
			}
			return &ReadOutput{IsCached: true, Instances: instances}
		}
	}
	return nil
}

func describeInstances(config *Config) map[string]*Instance {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:            aws.Config{Region: aws.String(config.Region)},
		SharedConfigState: session.SharedConfigEnable,
	}))
	ec2Svc := ec2.New(sess)

	describedInstancesOutput, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	instances := outputToInstances(describedInstancesOutput)

	describedVpcsOutput, err := ec2Svc.DescribeVpcs(nil)
	if err != nil {
		log.Fatal(err)
	}
	vpcs := outputToVpcs(describedVpcsOutput)

	return Merge(instances, vpcs)
}

func writeCache(tmpDir string, fileName string, v interface{}) {
	bytes, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	tmpFile := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(tmpFile, bytes, 0666); err != nil {
		log.Fatal(err)
	}
}

func outputToInstances(output *ec2.DescribeInstancesOutput) map[string]*Instance {
	instances := make(map[string]*Instance)
	ec2Instances := flattenEc2Instances(output)
	for _, ec2Instance := range ec2Instances {
		name := ""
		tags := make(map[string]string)
		var roles []string
		for _, tag := range ec2Instance.Tags {
			tagKey := aws.StringValue(tag.Key)
			tagValue := aws.StringValue(tag.Value)
			if tagKey == "Name" {
				name = tagValue
			}
			if tagKey == "Role" {
				roles = strings.Split(tagValue, ",")
			}
			tags[tagKey] = tagValue
		}
		instanceId := aws.StringValue(ec2Instance.InstanceId)
		instances[instanceId] = &Instance{
			InstanceId: instanceId,
			Tags:       tags,
			Name:       name,
			Roles:      roles,
			VpcId:      aws.StringValue(ec2Instance.VpcId),
		}
	}
	return instances
}

func flattenEc2Instances(output *ec2.DescribeInstancesOutput) []*ec2.Instance {
	var ec2Instances []*ec2.Instance
	for _, reservation := range output.Reservations {
		for _, ec2Instance := range reservation.Instances {
			ec2Instances = append(ec2Instances, ec2Instance)
		}
	}
	return ec2Instances
}

func outputToVpcs(output *ec2.DescribeVpcsOutput) map[string]*Vpc {
	vpcs := make(map[string]*Vpc)
	for _, output := range output.Vpcs {
		name := ""
		shortName := ""
		tags := make(map[string]string)
		for _, tag := range output.Tags {
			tagKey := aws.StringValue(tag.Key)
			tagValue := aws.StringValue(tag.Value)
			if tagKey == "Name" {
				name = tagValue
			}
			if tagKey == "ShortName" {
				shortName = tagValue
			}
			tags[tagKey] = tagValue
		}
		vpcId := aws.StringValue(output.VpcId)
		vpcs[vpcId] = &Vpc{
			VpcId:     vpcId,
			Tags:      tags,
			Name:      name,
			ShortName: shortName,
		}
	}
	return vpcs
}
