package main

//TCP is a the pipline allow anything in and out
//HTTP is a protocal that defines how data is formatted and transmitted over the web

import (
	"fmt"
	"net"
	"strings"
)

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
}

func main() {
	// Listen for incoming connections on port 8080
	listener, err := net.Listen("tcp", "localhost:8080")

	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return //exit if error
	}

	for {
		conn, err := listener.Accept() //handles handshake

		if err != nil {
			fmt.Println("Error accepting connection:", err)
		}

		fmt.Println("Client connected")

		go handleConnection(conn) //handle the connection in a separate function
	}

}

func handleConnection(conn net.Conn) {
	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)

		if err != nil {
			fmt.Println("Error reading from connection:", err)
			conn.Close()
			return
		}

		request := string(buffer[:n])

		fmt.Printf("Received: %s\n", request)

		req := parseRequest(request)

		fmt.Printf("Method: %s, Path: %s, Version: %s\n", req.Method, req.Path, req.Version)
		for key, value := range req.Headers {
			fmt.Printf("Header: %s: %s\n", key, value)
		}

		response := responseMessage(req)

		conn.Write([]byte(response))

	}
}

func parseRequest(request string) Request {
	//splits the request into lines
	requestLines := strings.Split(request, "\r\n")

	parts := strings.Split(requestLines[0], " ")

	method, path, version := parts[0], parts[1], parts[2]

	req := Request{
		Method:  method,
		Path:    path,
		Version: version,
		Headers: make(map[string]string),
	}

	for i := 1; i < len(requestLines); i++ {
		// An empty line indicates the end of headers
		if requestLines[i] == "" {
			break
		}
		line := requestLines[i]

		//SplitN so only 2 parts incase header value contains ": "
		split := strings.SplitN(line, ": ", 2)
		key, value := split[0], split[1]
		req.Headers[key] = value
	}

	return req
}

func responseMessage(req Request) string {
	//variable to exist outside of switch statement
	var message string
	var status string

	switch req.Path {
	case "/":
		//handle root path
		message = "hello "
		status = "200 OK"
	case "/about":
		//handle about path
		message = "about page"
		status = "200 OK"
	default:
		//handle 404 not found
		message = "404 not found"
		status = "404 Not Found"
	}

	//response back to client after req
	response := fmt.Sprintf(
		"%s %s\r\nContent-Length: %d\r\n\r\n%s",
		req.Version, status, len(message), message,
	)
	return response
}
