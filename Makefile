Version := $(shell git describe --tags --dirty)
GitCommit := $(shell git rev-parse HEAD)
LDFLAGS := "-s -w -X 'main.Version=$(Version)' -X 'main.GitCommit=$(GitCommit)'"
PLATFORM := $(shell ./hack/platform-tag.sh)
SOURCE_DIRS = main.go
export GO111MODULE=on

.PHONY: all
all: gofmt test build dist hash

.PHONY: build
build:
	go build -ldflags $(LDFLAGS)

.PHONY: gofmt
gofmt:
	@test -z $(shell gofmt -l -s $(SOURCE_DIRS) ./ |grep -v vendor/| tee /dev/stderr) || (echo "[WARN] Fix formatting issues with 'make gofmt'" && exit 1)

.PHONY: test
test:
	CGO_ENABLED=0 go test $(shell go list ./... | grep -v /vendor/|xargs echo) -cover

.PHONY: dist
dist:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bin/gha-bump
	CGO_ENABLED=0 GOOS=darwin go build -ldflags $(LDFLAGS) -o bin/gha-bump-darwin
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -ldflags $(LDFLAGS) -o bin/gha-bump-darwin-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags $(LDFLAGS) -o bin/gha-bump-armhf
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags $(LDFLAGS) -o bin/gha-bump-arm64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bin/gha-bump.exe

.PHONY: hash
hash:
	rm -rf bin/*.sha256 && ./hack/hashgen.sh
