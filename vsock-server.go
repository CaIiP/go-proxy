package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mdlayher/vsock"
)

const (
	// CID 16 for the enclave
	enclaveCID = uint32(16)
	// Port to listen on within the Nitro Enclave
	vsockPort = uint32(8080)
)

func main() {
	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetPrefix("[nitro-enclave-server] ")
	log.Printf("Starting Nitro Enclave Go Server on CID %d, port %d", enclaveCID, vsockPort)

	// Create a VSOCK listener
	l, err := vsock.ListenContextID(enclaveCID, vsockPort, nil)
	if err != nil {
		log.Fatalf("Failed to create vsock listener: %v", err)
	}
	defer l.Close()

	log.Printf("VSOCK server listening on CID %d port %d", enclaveCID, vsockPort)

	// Handle shutdown gracefully
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Channel to communicate server errors
	errChan := make(chan error)

	// Start accepting connections in a goroutine
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				errChan <- fmt.Errorf("failed to accept connection: %v", err)
				return
			}

			// Handle each connection in a new goroutine
			go handleHTTPConnection(conn)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-shutdown:
		log.Println("Shutdown signal received, closing server...")
	case err := <-errChan:
		log.Printf("Server error: %v", err)
	}

	log.Println("Server shutdown complete")
}

func handleHTTPConnection(conn net.Conn) {
	defer conn.Close()

	// Use a buffered reader to read HTTP request
	reader := bufio.NewReader(conn)

	// Read the first line to get the request method and path
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading request line: %v", err)
		return
	}

	// Parse the request line
	parts := strings.Fields(requestLine)
	if len(parts) < 3 {
		log.Printf("Invalid HTTP request line: %s", requestLine)
		return
	}
	method := parts[0]
	path := parts[1]

	// Read headers until we hit an empty line
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading header: %v", err)
			return
		}

		// Remove trailing CRLF
		line = strings.TrimRight(line, "\r\n")

		// Empty line marks end of headers
		if line == "" {
			break
		}

		// Parse header
		colonIndex := strings.Index(line, ":")
		if colonIndex > 0 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])
			headers[key] = value
		}
	}

	// Check if there's a request body
	var body string
	if contentLength, exists := headers["Content-Length"]; exists {
		// If there's content, read it
		// Note: This is a simplified approach and may not handle all HTTP cases
		bodyBytes := make([]byte, 0)
		if cl, err := fmt.Sscanf(contentLength, "%d", &bodyBytes); err == nil && cl > 0 {
			bodyBytes = make([]byte, cl)
			_, err = reader.Read(bodyBytes)
			if err != nil {
				log.Printf("Error reading body: %v", err)
			}
			body = string(bodyBytes)
		}
	}

	// Log the request details
	log.Printf("Received HTTP %s request for %s from %s", method, path, conn.RemoteAddr())
	if body != "" {
		log.Printf("Request body: %s", body)
	}

	// Prepare the HTTP response
	jsonResponse := `{"status":"success","message":"Request acknowledged by Nitro Enclave"}`
	httpResponse := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
		"Content-Type: application/json\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n"+
		"%s",
		len(jsonResponse), jsonResponse)

	// Send the HTTP response
	_, err = conn.Write([]byte(httpResponse))
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}