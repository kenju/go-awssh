package main

import (
	"io/fs"
	"reflect"
	"testing"
	"time"
)

// TempfsFileInfoMock implements fs.FileInfo
// and mocks Tempfs behaviour
type TempfsFileInfoMock struct {
	MockName    string
	MockModTime time.Time
}

func (tmpfs TempfsFileInfoMock) Name() string {
	if len(tmpfs.MockName) == 0 {
		return "tmp123.txt"
	}
	return tmpfs.MockName
}

func (tmpfs TempfsFileInfoMock) Size() int64 {
	return 10
}

func (tmpfs TempfsFileInfoMock) Mode() fs.FileMode {
	return fs.ModeTemporary
}

func (tmpfs TempfsFileInfoMock) ModTime() time.Time {
	if tmpfs.MockModTime.IsZero() {
		return time.Now()
	}
	return tmpfs.MockModTime
}

func (tmpfs TempfsFileInfoMock) IsDir() bool {
	return false
}

func (tmpfs TempfsFileInfoMock) Sys() interface{} {
	return nil
}

// DirFileInfoMock implements fs.FileInfo
// and mocks a directory entry behaviour
type DirFileInfoMock struct {
	MockName    string
	MockModTime time.Time
}

func (dir DirFileInfoMock) Name() string {
	if len(dir.MockName) == 0 {
		return "foo/"
	}
	return dir.MockName
}

func (dir DirFileInfoMock) Size() int64 {
	return 0
}

func (dir DirFileInfoMock) Mode() fs.FileMode {
	return fs.ModeDir
}

func (dir DirFileInfoMock) ModTime() time.Time {
	if dir.MockModTime.IsZero() {
		return time.Now()
	}
	return dir.MockModTime
}

func (dir DirFileInfoMock) IsDir() bool {
	return true
}

func (dir DirFileInfoMock) Sys() interface{} {
	return nil
}

func TestInstanceFdn(t *testing.T) {
	var tests = []struct {
		testName string
		instance *Instance
		config   *Config
		expected string
	}{
		{
			"default value",
			&Instance{},
			&Config{},
			"",
		},
		{
			"with InstanceId",
			&Instance{
				InstanceId: "foo123",
			},
			&Config{
				TemplateFqdn: "{{.InstanceId}}.aws.example.com",
			},
			"foo123.aws.example.com",
		},
		{
			"with InstanceId and VpcId",
			&Instance{
				InstanceId: "foo123",
				VpcId:      "bar456",
			},
			&Config{
				TemplateFqdn: "{{.InstanceId}}.{{.VpcId}}.aws.example.com",
			},
			"foo123.bar456.aws.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			actual := tt.instance.Fqdn(tt.config)
			if tt.expected != actual {
				t.Errorf("expected=%s, actual=%s\n", tt.expected, actual)
			}
		})
	}
}

func TestReadOutputBuildCandidates(t *testing.T) {
	var tests = []struct {
		testName   string
		readOutput *ReadOutput
		expected   string
	}{
		{
			"default value",
			&ReadOutput{},
			"",
		},
		{
			"with one Instance",
			&ReadOutput{
				Instances: map[string]*Instance{
					"fooId": &Instance{
						InstanceId: "fooId",
						Name:       "fooName",
						Vpc: &Vpc{
							Name: "fooVpcName",
						},
					},
				},
			},
			"fooName fooVpcName fooId",
		},
		{
			"with two Instance",
			&ReadOutput{
				Instances: map[string]*Instance{
					"fooId": &Instance{
						InstanceId: "fooId",
						Name:       "fooName",
						Vpc: &Vpc{
							Name: "fooVpcName",
						},
					},
					"barId": &Instance{
						InstanceId: "barId",
						Name:       "barName",
						Vpc: &Vpc{
							Name: "barVpcName",
						},
					},
				},
			},
			"barName barVpcName barId\n" +
				"fooName fooVpcName fooId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			actual := tt.readOutput.BuildCandidates()
			if tt.expected != actual {
				t.Errorf("expected=%s, actual=%s\n", tt.expected, actual)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	var tests = []struct {
		testName  string
		instances map[string]*Instance
		vpcs      map[string]*Vpc
		expected  map[string]*Instance
	}{
		{
			"default value",
			map[string]*Instance{},
			map[string]*Vpc{},
			map[string]*Instance{},
		},
		{
			"matches from one vpc",
			map[string]*Instance{
				"fooInstanceId": &Instance{
					VpcId: "fooVpcId",
				},
			},
			map[string]*Vpc{
				"fooVpcId": &Vpc{
					VpcId: "fooVpcId",
					Name:  "fooVpcName",
				},
				"barVpcId": &Vpc{
					VpcId: "barVpcId",
					Name:  "barVpcName",
				},
			},
			map[string]*Instance{
				"fooInstanceId": &Instance{
					VpcId: "fooVpcId",
					Vpc: &Vpc{
						VpcId: "fooVpcId",
						Name:  "fooVpcName",
					},
				},
			},
		},
		{
			"multiple matches",
			map[string]*Instance{
				"fooInstanceId": &Instance{
					VpcId: "fooVpcId",
				},
				"barInstanceId": &Instance{
					VpcId: "barVpcId",
				},
			},
			map[string]*Vpc{
				"fooVpcId": &Vpc{
					VpcId: "fooVpcId",
					Name:  "fooVpcName",
				},
				"barVpcId": &Vpc{
					VpcId: "barVpcId",
					Name:  "barVpcName",
				},
			},
			map[string]*Instance{
				"fooInstanceId": &Instance{
					VpcId: "fooVpcId",
					Vpc: &Vpc{
						VpcId: "fooVpcId",
						Name:  "fooVpcName",
					},
				},
				"barInstanceId": &Instance{
					VpcId: "barVpcId",
					Vpc: &Vpc{
						VpcId: "barVpcId",
						Name:  "barVpcName",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			actual := Merge(tt.instances, tt.vpcs)
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected=%+v, actual=%+v\n", tt.expected, actual)
			}
		})
	}
}

func TestIsCacheEntry(t *testing.T) {
	var tests = []struct {
		testName string
		entry    fs.FileInfo
		prefix   string
		expected bool
	}{
		{
			"tmpfs with empty prefix",
			TempfsFileInfoMock{},
			"",
			false,
		},
		{
			"directory entry with empty prefix",
			DirFileInfoMock{},
			"",
			true,
		},
		{
			"tmpfs with matched prefix",
			TempfsFileInfoMock{MockName: "foo-bar-123"},
			"foo-bar",
			false,
		},
		{
			"directory entry with matched prefix",
			DirFileInfoMock{MockName: "foo-bar-123"},
			"foo-bar",
			true,
		},
		{
			"tmpfs with unmatched prefix",
			TempfsFileInfoMock{MockName: "foo-bar-123"},
			"buz",
			false,
		},
		{
			"directory entry with unmatched prefix",
			DirFileInfoMock{MockName: "foo-bar-123"},
			"buz",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			actual := IsCacheEntry(tt.entry, tt.prefix)
			if tt.expected != actual {
				t.Errorf("expected=%t, actual=%t\n", tt.expected, actual)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	var tests = []struct {
		testName string
		entry    fs.FileInfo
		expected bool
	}{
		{
			"tmpfs within expiresIn",
			TempfsFileInfoMock{},
			false,
		},
		{
			"directory within expiresIn",
			DirFileInfoMock{},
			false,
		},
		{
			"tmpfs with out range of expiresIn",
			TempfsFileInfoMock{
				MockModTime: func() time.Time {
					return time.Now().AddDate(0, 0, -2)
				}(),
			},
			true,
		},
		{
			"directory with out range of expiresIn",
			DirFileInfoMock{
				MockModTime: func() time.Time {
					return time.Now().AddDate(0, 0, -2)
				}(),
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			actual := IsExpired(tt.entry)
			if tt.expected != actual {
				t.Errorf("expected=%t, actual=%t\n", tt.expected, actual)
			}
		})
	}
}
