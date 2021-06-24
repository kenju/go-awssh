# go-awssh

[![Go Reference](https://pkg.go.dev/badge/github.com/kenju/go-awssh.svg)](https://pkg.go.dev/github.com/kenju/go-awssh) [![ci-test](https://github.com/kenju/go-awssh/actions/workflows/ci-test.yml/badge.svg)](https://github.com/kenju/go-awssh/actions/workflows/ci-test.yml)

`go-awssh` is a developer tool to make your SSH to AWS EC2 instances easy.

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

```
Usage of go-awssh:
  -purge-cache
        purge local cache of AWS API calls
  -region string
        AWS Region for the session (default "us-east-1")
  -ssh-bin string
        a path to the binary for SSH (default "ssh")
  -template-fqdn string
        a template for building FQDN for servers based on text/template interface (default "{{.Name}}.aws.example.com")
```

## Examples

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

## Pro Tips

### Accessing multiple servers

You can make use of [`cssh(1)`](https://linux.die.net/man/1/cssh) and SSH to multiple servers at the same time. Once you install `cssh`, pass `-ssh-bin` as follows:

```
/path/to/go-awssh \
    -region ap-northeast-1 \
    -ssh-bin cssh \
    -template-fqdn "{{.InstanceId}}.aws.yourdomain.com"
```

`peco` allows you to select multiple lines by [StickySelection](https://github.com/peco/peco#stickyselection) feature.

[`peco.ToggleSelectionAndSelectNext`](https://github.com/peco/peco#available-actions) allows you to select the current line, saves it, and proceeds to the next line.

```json
{
    "Keymap": {
        "C-t": "peco.ToggleSelectionAndSelectNext"
    },
    "StickySelection": true
}
```

## Development

### Unit Tests

Run `make test` locally.

[GitHub Actions](https://github.com/kenju/go-awssh/actions/workflows/ci-test.yml) runs when commits pushed.

### Release

`git tag` and push to the `master` branch.

[`goreleaser`](https://goreleaser.com/) is triggered via [GitHub Actions](https://github.com/kenju/go-awssh/actions/workflows/release.yml).
