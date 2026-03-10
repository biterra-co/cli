# Biterra CLI — development targets
# Run from biterra-cli/

.PHONY: all build test lint vet clean install

BINARY := biterra
ifeq ($(OS),Windows_NT)
	BINARY := biterra.exe
endif

# Default: run tests
all: test

build: vet
	@go build -o $(BINARY) .

test:
	@go test -v -count=1 ./...

vet:
	@go vet ./...

lint: vet
	@go test -count=1 ./...
	@echo "Consider: golangci-lint run (if installed)"

clean:
	@rm -f $(BINARY) coverage.out coverage.html

install: build
	@go install .
