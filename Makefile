.PHONY: build test vet ci run clean build-windows build-linux

BINARY := bin/photomatch
CMD := ./cmd/photomatch
VERSION ?= 0.1.0
LDFLAGS := -X main.version=$(VERSION)

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

ci: vet test build

run:
	go run -ldflags "$(LDFLAGS)" $(CMD) -config config.yaml

build-windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/photomatch.exe $(CMD)

build-linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/photomatch $(CMD)

clean:
	rm -rf bin
