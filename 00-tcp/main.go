package main

/*
Why?
Data is split into packets and send through the web
Without TCP, packets would be out of order and get lost
With TCP, rules are set to ensure data arrives safely and put in the right order.

1. handshake: client and server exchange messages to establish a connection (3 way handshake)
2. Page numbers: each packet is given a number so they can be put in the right order
3. acknowledgement: the receiver sends back a message to the sender to confirm that the packet was received. If the sender doesn't receive an acknowledgement, it will resend the packet.

https://pkg.go.dev/net
*/

import (
	"fmt"
	"net"
)

func main() {
	// Listen for incoming connections on port 8080
	listener, err := net.Listen("tcp", "localhost:8080")

	if err != nil {
		fmt.Println("Error starting TCP server:", err)
	}

	//Loop to read data from the connection
	for {

		// Accept incoming connections allowing reading and writing to the connection
		conn, err := listener.Accept() //handles handshake

		if err != nil {
			fmt.Println("Error accepting connection:", err)
		}

		fmt.Println("Client connected")

		//Go allows concurrent handling using goroutines
		go handleConnection(conn) //handle the connection in a separate function
	}

}

func handleConnection(conn net.Conn) {
	for {
		buffer := make([]byte, 1024) //buffer to hold incoming data
		//  buffer = [h, e, l, l, o, 0, 0, 0, 0, 0, ...]
		n, err := conn.Read(buffer)

		if err != nil {
			fmt.Println("Error reading from connection:", err)
			conn.Close() //when they close conenction ( it gives EOF) so we should release the resources
			return
		}
		fmt.Printf("Received: %s\n", string(buffer[:n]))
	}
}
