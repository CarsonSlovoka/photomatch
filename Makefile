.PHONY: build test vet cover cover-view ci run clean build-windows build-linux build-mac

BINARY := bin/photomatch
CMD := ./cmd/photomatch
VERSION ?= 0.1.0
LDFLAGS := -X main.version=$(VERSION)
COVER_OUT ?= coverage.out
COVER_MIN ?= 60

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

# 語句覆蓋率。低於 COVER_MIN（預設 60）則失敗。可覆寫：make cover COVER_MIN=70
cover:
	go test -count=1 -coverprofile=$(COVER_OUT) -covermode=atomic ./...
	go tool cover -func=$(COVER_OUT)
	@total=$$(go tool cover -func=$(COVER_OUT) | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
	echo "total coverage: $${total}% (minimum $(COVER_MIN)%)"; \
	awk -v total="$$total" -v min="$(COVER_MIN)" 'BEGIN { if ((total + 0) < (min + 0)) exit 1 }' \
		|| { echo "coverage $${total}% is below minimum $(COVER_MIN)%"; exit 1; }
	go tool cover -html=$(COVER_OUT) -o coverage.html

cover-view:
	go tool cover -html=$(COVER_OUT)

ci: vet cover build

run:
	go run -ldflags "$(LDFLAGS)" $(CMD) -config config.yaml

build-windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/photomatch.exe $(CMD)

build-linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/photomatch $(CMD)

# Apple Silicon（darwin/arm64）(不建 Intel)
build-mac:
	mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/photomatch-darwin-arm64 $(CMD)

clean:
	rm -rf bin
