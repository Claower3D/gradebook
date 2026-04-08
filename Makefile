.PHONY: build run dev deps clean

# Build binary
build:
	cd cmd/server && go build -o ../../bin/gradebook .

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run dev server
run: build
	./bin/gradebook

# Run directly (no build step)
dev:
	cd cmd/server && go run .

# Clean
clean:
	rm -rf bin/

# Build for Linux (production)
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
	cd cmd/server && go build -o ../../bin/gradebook-linux .
