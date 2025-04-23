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