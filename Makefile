VERSION ?= 0.1.0
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"
PKG     := ./cmd/tally

.PHONY: build test clean build-all checksums

build:
	go build $(LDFLAGS) -o bin/tally $(PKG)

test:
	go test ./...

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/tally-windows-amd64.exe $(PKG)

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/tally-darwin-arm64 $(PKG)

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/tally-darwin-amd64 $(PKG)

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/tally-linux-amd64 $(PKG)

build-all: build-windows build-darwin-arm64 build-darwin-amd64 build-linux-amd64

checksums:
	cd bin && shasum -a 256 tally-* > checksums.txt

clean:
	rm -f bin/tally bin/tally-* bin/checksums.txt
