// control the number of requests a client can make in a given time frame
// implement token bucket algo how
package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type TokenBucket struct {
	capacity       int
	refillRate     int
	tokens         int
	lastRefillTime int64
}

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
}

func NewTokenBucket(capacity int,
	refillRate int) TokenBucket {
	return TokenBucket{
		capacity:       capacity,
		refillRate:     refillRate,
		tokens:         capacity,
		lastRefillTime: time.Now().Unix(),
	}
}
func (r *TokenBucket) Allow() bool { //pointer to modify orignal bucket

	//see how many tokens to add based on time elapsed
	now := time.Now().Unix()
	elapsed := now - r.lastRefillTime
	r.tokens = min(r.capacity, r.tokens+int(elapsed)*r.refillRate)
	r.lastRefillTime = now

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

func main() {
	listener, err := net.Listen("tcp", "localhost:8080")

	var mu sync.Mutex //mutex to protect shared access to ipLimiter

	//create token bucket here to persist across connections
	ipLimiter := make(map[string]*TokenBucket)

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

		//pass exact mutex so all connection lock and unlock on the same one
		go handleConnection(conn, &mu, ipLimiter)

	}

}

func handleConnection(conn net.Conn, mu *sync.Mutex, ipLimiter map[string]*TokenBucket) {

	mu.Lock() //lock to safely access ipLimiter map
	ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		fmt.Println("Error getting client IP:", err)
		conn.Close()
		return
	}
	bucket, exists := ipLimiter[ip]
	if !exists {
		fmt.Println("Creating new token bucket for IP:", ip)
		b := NewTokenBucket(5, 1) //capacity of 5 tokens and refill rate of 1 token per second
		bucket = &b               //pass pointer to bucket so we can modify it in the handler
		ipLimiter[ip] = bucket
	}
	mu.Unlock()

	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)

		if err != nil {
			fmt.Println("Error reading from connection:", err)
			conn.Close()
			return
		}

		//lock mutex to safely check incase 2 connection from same IP at same time
		mu.Lock()
		allowed := bucket.Allow() //check token bucket if enough tokens
		mu.Unlock()

		if !allowed {
			conn.Write([]byte("HTTP/1.1 429 Too Many Requests\r\n\r\nRate limit exceeded\n"))
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
