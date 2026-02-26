.PHONY: build run test lint fmt tidy prettier deps clean

BINARY := bin/app

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/app

run: build
	$(BINARY)

test:
	go test -v ./...

lint:
	go vet ./...
	golangci-lint run

fmt:
	go fmt ./...

tidy:
	go mod tidy

prettier:
	prettier --write .

clean:
	rm -rf bin/
