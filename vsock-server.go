package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
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
	// Note: The actual mdlayher/vsock API might have changed, this should use the current API
	l, err := vsock.ListenContextID(enclaveCID, vsockPort)
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
			go handleConnection(conn)
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Log connection details
	log.Printf("Handling connection from %s", conn.RemoteAddr())

	// Simple protocol: Read request and send acknowledgement
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from connection: %v", err)
		return
	}

	log.Printf("Received %d bytes: %s", n, buffer[:n])

	// Send acknowledgement response
	response := []byte(`{"status":"success","message":"Request acknowledged by Nitro Enclave"}`)
	_, err = conn.Write(response)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}