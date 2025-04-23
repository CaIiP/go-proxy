package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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