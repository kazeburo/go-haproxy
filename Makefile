VERSION=0.0.2
GITCOMMIT?=$(shell git describe --dirty --always)
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.commit=${GITCOMMIT}"
all: haproxy-status-cli

.PHONY: haproxy-status-cli

haproxy-status-cli: cmd/haproxy-status-cli/*.go
	go build $(LDFLAGS) -o haproxy-status-cli cmd/haproxy-status-cli/*.go

linux: cmd/haproxy-status-cli/*.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o haproxy-status-cli cmd/haproxy-status-cli/*.go

check:
	go test -v ./...


