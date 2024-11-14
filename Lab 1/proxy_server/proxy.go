package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

const maxConcurrentProxyRequests = 10
var proxySemaphore = make(chan struct{}, maxConcurrentProxyRequests)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: proxy_server <port>")
		os.Exit(1)
	}
	port := os.Args[1]

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Error starting proxy server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Proxy server listening on port %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Limit concurrent requests
		proxySemaphore <- struct{}{}
		go handleProxyConnection(conn)
	}
}

func handleProxyConnection(clientConn net.Conn) {
	defer clientConn.Close()
	defer func() { <-proxySemaphore }()

	clientRequest, err := http.ReadRequest(bufio.NewReader(clientConn))
	if err != nil {
		writeProxyResponse(clientConn, http.StatusBadRequest, "text/plain", "Bad Request")
		return
	}

	// only allow GET requests
	if clientRequest.Method != http.MethodGet {
		writeProxyResponse(clientConn, http.StatusNotImplemented, "text/plain", "Not Implemented")
		return
	}

	targetResponse, err := forwardRequest(clientRequest)
	if err != nil {
		writeProxyResponse(clientConn, http.StatusBadGateway, "text/plain", "Bad Gateway")
		return
	}
	defer targetResponse.Body.Close()

	// relay response back to client
	relayResponse(clientConn, targetResponse)
}

func forwardRequest(clientRequest *http.Request) (*http.Response, error) {
	targetRequest, err := http.NewRequest(clientRequest.Method, clientRequest.RequestURI, nil)
	if err != nil {
		return nil, err
	}

	// copy the headers
	for name, values := range clientRequest.Header {
		for _, value := range values {
			targetRequest.Header.Add(name, value)
		}
	}

	// send the request
	client := &http.Client{}
	return client.Do(targetRequest)
}

func relayResponse(clientConn net.Conn, targetResponse *http.Response) {
	targetResponse.Write(clientConn)
}

func writeProxyResponse(conn net.Conn, statusCode int, contentType, body string) {
	response := &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	response.Header.Set("Content-Type", contentType)
	response.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	response.Write(conn)
}
