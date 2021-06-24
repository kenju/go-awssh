.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	go build -o build/go-awssh

