// unlike HTTPS where it is non-real-time communication as in client request data when needed
// websocket allows real-time communication between client and server, mainting a connection open for continuous data exchange
package main

import (
	"crypto/sha1"
	"encoding/base64"
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
	listener, err := net.Listen("tcp", "localhost:8080")

	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println("Error accepting connection:", err)
		}
		fmt.Println("Client connected")
		go handleConnection(conn)

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

		for key, value := range req.Headers {
			//check if client requesting upgrade to websocket
			if key == "Upgrade" && value == "websocket" {
				fmt.Println("Websocket connection established")

				//grab client key to generate accept key
				clientKey := req.Headers["Sec-WebSocket-Key"]
				handleWebsocket(conn, clientKey)
				return //stop handling HTTP
			}
		}

		//conn.Write([]byte(response))

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

func handleWebsocket(conn net.Conn, key string) {
	//SHA1 hash of client key + magic string, then base64 encode the result to get the accept key
	hasher := sha1.New()
	hasher.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	sha1Bytes := hasher.Sum(nil)
	acceptKey := base64.StdEncoding.EncodeToString(sha1Bytes)

	conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"))

	//loop through websocket frames
	for {
		//data is passed in specific format and need to decode it according to the websocket protocol

		//alocate 2 empty byte slots for incom data
		header := make([]byte, 2)
		conn.Read(header) //read 2 bytes from TCP connection.

		//isolate the opcode, payload length, and masked bit from the header
		opcode := header[0] & 0x0F
		payloadLen := header[1] & 0x7F
		masked := header[1] & 0x80

		//close connection (if opcode is 8, it means the client is closing the connection)
		if opcode == 8 {
			fmt.Println("Websocket connection closed by client")
			conn.Close()
			return
		}

		mask := make([]byte, 4)

		if masked != 0 {
			conn.Read(mask) //read the 4 byte mask from the connection
		}

		payload := make([]byte, payloadLen)
		conn.Read(payload) //read the payload data from the connection

		//unmask the payload data using the mask
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
		fmt.Println(string(payload))

		//write back repsonse
		// msg := []byte("goodbye") if you want to put your own

		//encode pre data
		conn.Write([]byte{0x81,
			byte(len(payload))})

		//write payload data
		conn.Write(payload)

	}
}
