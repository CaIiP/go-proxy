#!/bin/bash
set -e

echo "Preparing application ZIP files..."

# Create temporary directories
mkdir -p tmp/ec2-client
mkdir -p tmp/enclave-server
mkdir -p tmp/viproxy

# Create client application file
cat > tmp/ec2-client/client.go << 'EOL'
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	log.Println("EC2 Client Application starting...")

	// Get proxy endpoint from environment or use default
	proxyEndpoint := "http://127.0.0.1:8000"
	if endpoint := os.Getenv("PROXY_ENDPOINT"); endpoint != "" {
		proxyEndpoint = endpoint
	}

	// Continuously send requests to the proxy
	for {
		response, err := sendRequest(proxyEndpoint)
		if err != nil {
			log.Printf("Error sending request: %v", err)
		} else {
			log.Printf("Response from enclave: %s", response)
		}

		time.Sleep(5 * time.Second)
	}
}

func sendRequest(endpoint string) (string, error) {
	log.Printf("Sending request to %s", endpoint)

	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}
EOL

# Create server application file
cat > tmp/enclave-server/server.go << 'EOL'
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	log.Println("Nitro Enclave Server Application starting...")

	// Get server port from environment or use default
	port := "8000"
	if p := os.Getenv("SERVER_PORT"); p != "" {
		port = p
	}

	// Register handler for root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request from %s", r.RemoteAddr)

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		response := fmt.Sprintf("Hello from Nitro Enclave! (Host: %s, Time: %s)",
			hostname,
			fmt.Sprintf("%v", time.Now().Format(time.RFC3339)))

		log.Printf("Sending response: %s", response)
		fmt.Fprint(w, response)
	})

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
EOL

# Create Dockerfile for enclave
cat > tmp/enclave-server/Dockerfile << 'EOL'
FROM golang:1.19-alpine AS builder

# Set working directory
WORKDIR /app

# Copy source code
COPY server.go .

# Build the Go application
RUN go build -o server server.go

# Use a minimal alpine image for the final container
FROM alpine:3.17

# Install necessary packages
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/server /usr/local/bin/

# Expose the port
EXPOSE 8000

# Set the entry point
ENTRYPOINT ["/usr/local/bin/server"]
EOL

# Copy the viproxy code
cp -r path/to/viproxy/* tmp/viproxy/

# Create ZIP files
cd tmp
zip -r ec2-client.zip ec2-client/
zip -r enclave-server.zip enclave-server/
zip -r viproxy.zip viproxy/

# Move ZIP files to current directory
mv *.zip ../

# Clean up
cd ..
rm -rf tmp

echo "ZIP files created: ec2-client.zip, enclave-server.zip, viproxy.zip"
echo "Upload these files to your S3 bucket before deploying to EC2"