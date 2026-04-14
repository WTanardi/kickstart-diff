.PHONY: build install clean test release

# Build the binary
build:
	go build -o kickstart-diff .

# Install to $GOPATH/bin
install:
	go install

# Clean build artifacts
clean:
	rm -f kickstart-diff
	rm -rf dist/

# Run tests
test:
	go test -v ./...

# Build for all platforms (requires goreleaser)
release:
	goreleaser release --clean

# Build snapshot without publishing
snapshot:
	goreleaser release --snapshot --clean

# Run the tool
run:
	go run . ksync

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run
