# go-awssh

[![ci-test](https://github.com/kenju/go-awssh/actions/workflows/ci-test.yml/badge.svg)](https://github.com/kenju/go-awssh/actions/workflows/ci-test.yml)

`go-awssh` is a developer tool to make your SSH easy to AWS EC2 instances.

Describing Instances/VPCs data, select one or multiple instances, and make connection(s) to selected instances. Caching the response of API calls for 1day using Tmpfs.

## Requirements

- your process has been granted to execute (IAM):
    - [AWS EC2 DescribeInstances](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstances.html)
    - [AWS EC2 DescribeVpcs](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpcs.html)
- `ssh` is installed and in your $PATH
    - or, alternative SSH command which is configured by `-ssh-bin`
- `peco` is installed and in your $PATH
    - https://github.com/peco/peco

## Usage

- `-purge-cache`
- `-region`
- `-ssh-bin`
- `-template-fqdn`

### Example A

Connect `<instance_id>.aws.yourdomain.com` via `ssh` command by retrieving AWS EC2 Instance/VPC data from `us-east-1` region:

```sh
/path/to/go-awssh \
    -region us-east-1 \
    -ssh-bin ssh \
    -template-fqdn "{{.InstanceId}}.aws.yourdomain.com"
```

### Example B

Connect `<instance_id>.aws.yourdomain.com` via `cssh` command by retrieving AWS EC2 Instance/VPC data from `ap-northeast-1` region:

```sh
/path/to/go-awssh \
    -region ap-northeast-1 \
    -ssh-bin cssh \
    -template-fqdn "{{.InstanceId}}.aws.yourdomain.com"
```
