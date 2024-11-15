# Lab 1

Group 51: Jonas Goh, Nicolas Carboni

## Server
The server is a basic HTTP server that handles GET and POST requests for files. Key features include:

- Concurrent request handling with a limit of 10 simultaneous connections
- Support for common file types (.html, .txt, .gif, .jpeg, .jpg, .css) 
- GET requests:
  - Returns file contents with appropriate content type if file exists
  - Returns 404 if file not found
  - Returns 400 for unsupported file types
- POST requests:
  - Creates/overwrites files with request body content
  - Returns 201 on successful creation
  - Returns 400 for unsupported file types
- Returns 501 Not Implemented for unsupported HTTP methods
- Basic error handling for malformed requests and server errors


## Proxy
The proxy server forwards HTTP GET requests from clients to the origin server. Key features include:

- Concurrent request handling with a limit of 10 simultaneous connections
- Only handles GET requests, returns 501 Not Implemented for other methods
- Forwards client request headers to origin server
- Relays origin server response back to client
- Basic error handling:
  - Returns 400 for malformed requests
  - Returns 502 Bad Gateway for failed origin server connections


## Test
The test script checks the server and proxy with various requests to ensure they handle different scenarios correctly. It includes:

- GET requests for existing and non-existing files
- POST requests to create new files
- Unsupported file types
- Unsupported HTTP methods

## Notes
The cloud implementation is not included.

[Github link to Lab 1](https://github.com/jonasgwt/tda596-labs/tree/main/Lab%201)
