VERSION ?= 0.1.0
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"
PKG     := ./cmd/tally
BIN     := skills/tally/bin

.PHONY: build test clean build-all checksums

build:
	go build $(LDFLAGS) -o $(BIN)/tally $(PKG)

test:
	go test ./...

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN)/tally-windows-amd64.exe $(PKG)

build-all: build-windows

checksums:
	cd $(BIN) && shasum -a 256 tally-* > checksums.txt

clean:
	rm -f $(BIN)/tally $(BIN)/tally-* $(BIN)/checksums.txt
