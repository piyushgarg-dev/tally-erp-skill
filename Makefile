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

build-all: build-windows

checksums:
	cd bin && shasum -a 256 tally-* > checksums.txt

clean:
	rm -f bin/tally bin/tally-* bin/checksums.txt
