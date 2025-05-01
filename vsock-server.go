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
	// Port to listen on within the Nitro Enclave
	vsockPort = uint32(8080)
)

func main() {
	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetPrefix("[nitro-enclave-server] ")
	log.Printf("Starting Nitro Enclave Go Server on vsock port %d", vsockPort)

	// Create a VSOCK listener
	// VM_VSOCK_CID_ANY (-1u or 0xFFFFFFFF) means listen for connections from any CID
	listener, err := vsock.ListenContextID(vsock.ContextIDHost, vsockPort, nil)
	if err != nil {
		log.Fatalf("Failed to create vsock listener: %v", err)
	}
	defer listener.Close()

	log.Printf("VSOCK server listening on port %d", vsockPort)

	// Handle shutdown gracefully
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Channel to communicate server errors
	errChan := make(chan error)

	// Start accepting connections in a goroutine
	go func() {
		for {
			conn, err := listener.Accept()
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

	// Log connection details if it's a vsock connection
	if vsockConn, ok := conn.(*vsock.Conn); ok {
		local, remote := vsockConn.LocalAddr(), vsockConn.RemoteAddr()
		vsockLocal, _ := local.(*vsock.Addr)
		vsockRemote, _ := remote.(*vsock.Addr)
		log.Printf("Handling connection from CID %d Port %d to CID %d Port %d",
			vsockRemote.ContextID, vsockRemote.Port,
			vsockLocal.ContextID, vsockLocal.Port)
	} else {
		log.Printf("Handling connection from %s", conn.RemoteAddr())
	}

	// Simple protocol: Read request and send acknowledgement
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from connection: %v", err)
		return
	}

	log.Printf("Received %d bytes", n)

	// Send acknowledgement response
	response := []byte(`{"status":"success","message":"Request acknowledged by Nitro Enclave"}`)
	_, err = conn.Write(response)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}