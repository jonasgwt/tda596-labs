package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxConcurrentRequests = 10
var semaphore = make(chan struct{}, maxConcurrentRequests)
var contentTypes = map[string]string{
	".html": "text/html",
	".txt":  "text/plain",
	".gif":  "image/gif",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".css":  "text/css",
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: http_server <port>")
		os.Exit(1)
	}
	port := os.Args[1]

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Server listening on port %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		// Limit concurrent requests
		semaphore <- struct{}{}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	defer func() { <-semaphore }()

	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		writeResponse(conn, http.StatusBadRequest, "text/plain", "Bad Request")
		return
	}

	switch request.Method {
	case http.MethodGet:
		handleGetRequest(conn, request)
	case http.MethodPost:
		handlePostRequest(conn, request)
	default:
		writeResponse(conn, http.StatusNotImplemented, "text/plain", "Not Implemented")
	}
}

func handleGetRequest(conn net.Conn, request *http.Request) {
	filePath := "." + request.URL.Path
	if !isAllowedFileType(filePath) {
		writeResponse(conn, http.StatusBadRequest, "text/plain", "Bad Request")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		writeResponse(conn, http.StatusNotFound, "text/plain", "Not Found")
		return
	}
	defer file.Close()

    fileContent, err := io.ReadAll(file)
    if err != nil {
        writeResponse(conn, http.StatusInternalServerError, "text/plain", "Internal Server Error")
        return
    }

    contentType := getContentType(filePath)
	writeResponse(conn, http.StatusOK, contentType, string(fileContent))
}

func handlePostRequest(conn net.Conn, request *http.Request) {
	filePath := "." + request.URL.Path
	if !isAllowedFileType(filePath) {
		writeResponse(conn, http.StatusBadRequest, "text/plain", "Bad Request")
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		writeResponse(conn, http.StatusInternalServerError, "text/plain", "Internal Server Error")
		return
	}
	defer file.Close()

	_, err = io.Copy(file, request.Body)
	if err != nil {
		writeResponse(conn, http.StatusInternalServerError, "text/plain", "Internal Server Error")
		return
	}

	writeResponse(conn, http.StatusCreated, "text/plain", "File created successfully")
}

func writeResponse(conn net.Conn, statusCode int, contentType, body string) {
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


func isAllowedFileType(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	_, allowed := contentTypes[ext]
	return allowed
}

func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if contentType, found := contentTypes[ext]; found {
		return contentType
	}
	log.Fatal("Unknown file type") // Should never happen
	return ""
}
