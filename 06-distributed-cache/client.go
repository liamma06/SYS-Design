package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// Client routes cache commands to the correct node using consistent hashing.
// The caller never needs to know which node holds a given key.
type Client struct {
	ring  *HashRing
	addrs map[string]string // node name → TCP address, e.g. "node-1" → "localhost:6379"
}

// create new clients when give a map of node names to addresses to create the hash ring
func NewClient(nodes map[string]string) *Client {
	ring := NewHashRing()
	for name := range nodes {
		ring.AddNode(name)
	}
	return &Client{ring: ring, addrs: nodes} //return pointer to new client
}

// sendCommand opens a TCP connection to the target node, sends the command, and returns the response.
func (cl *Client) sendCommand(key, command string) (string, error) {
	node := cl.ring.GetNode(key) //get node responsible for key
	addr := cl.addrs[node]       //get address of that node from the map

	conn, err := net.Dial("tcp", addr) //opebs connection to TCP connection
	if err != nil {
		return "", fmt.Errorf("could not connect to %s (%s): %w", node, addr, err)
	}
	defer conn.Close() //when func done close the connection

	fmt.Fprintf(conn, command+"\n")
	reader := bufio.NewReader(conn) //take input from user
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}
	return strings.TrimSpace(response), nil
}

// matches set of server to send in proper format
func (cl *Client) Set(key, value string, ttl time.Duration) error {
	var cmd string
	if ttl > 0 {
		cmd = fmt.Sprintf("SET %s %s %s", key, value, ttl)
	} else {
		cmd = fmt.Sprintf("SET %s %s", key, value)
	}
	resp, err := cl.sendCommand(key, cmd)
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	return nil
}

func (cl *Client) Get(key string) (string, bool) {
	resp, err := cl.sendCommand(key, fmt.Sprintf("GET %s", key))
	if err != nil || resp == "NULL" {
		return "", false
	}
	return resp, true
}

func (cl *Client) Delete(key string) error {
	resp, err := cl.sendCommand(key, fmt.Sprintf("DELETE %s", key))
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	return nil
}
