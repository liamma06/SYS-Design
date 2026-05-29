/*
A distributed cache is a key-value store spread across multiple servers.
This allows for larger storage capacity and faster access by distributing the load.
Challenges include:
- Data consistency: ensuring all servers have the same data
- Cache invalidation: deciding when to remove or update cached data
- Load balancing: distributing requests evenly across servers
*/
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Cache struct {
	mu    sync.RWMutex //mutex to protect concurrent access to the cache, ALL items share one
	items map[string]item
}

type item struct {
	value      string
	expiration time.Time
}

func (c *Cache) Set(key, value string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time

	if duration > 0 {
		exp = time.Now().Add(duration)
	}
	c.items[key] = item{value: value, expiration: exp}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return "", false
	}

	if item.expiration.IsZero() {
		return item.value, true
	}

	if time.Now().After(item.expiration) {
		return "", false
	}
	return item.value, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key) //delete is a built in function that removes the key from the map
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run . server <port>  OR  go run . client")
		return
	}

	switch os.Args[1] {
	case "server":
		port := "6379"
		if len(os.Args) >= 3 {
			port = os.Args[2]
		}
		runServer(port)

	case "client":
		runClientDemo()
	}
}

func runServer(port string) {
	c := &Cache{items: make(map[string]item)}
	c.Evict()

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	fmt.Println("Cache server listening on :" + port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn, c)
	}
}

func runClientDemo() {
	//create a client with 3 nodes in the cluster
	client := NewClient(map[string]string{
		"node-1": "localhost:6379",
		"node-2": "localhost:6380",
		"node-3": "localhost:6381",
	})

	// set some keys — each will be routed to a different node by the hash ring
	keys := []string{"user:1", "user:2", "order:99", "session:abc"}
	for _, key := range keys {
		err := client.Set(key, "value-of-"+key, 0)
		if err != nil {
			fmt.Printf("SET %s failed: %v\n", key, err)
		} else {
			fmt.Printf("SET %s → OK\n", key)
		}
	}

	fmt.Println()

	// get them back
	for _, key := range keys {
		val, ok := client.Get(key)
		if ok {
			fmt.Printf("GET %s → %s\n", key, val)
		} else {
			fmt.Printf("GET %s → not found\n", key)
		}
	}

	// set one with a TTL
	fmt.Println()
	client.Set("temp", "expires-soon", 3*time.Second)
	fmt.Println("SET temp with 3s TTL")
	time.Sleep(4 * time.Second)
	if _, ok := client.Get("temp"); !ok {
		fmt.Println("GET temp → expired, gone")
	}
}

func handleConnection(conn net.Conn, c *Cache) {
	reader := bufio.NewReader(conn) //bufio allows us to read input line by line instead of byte by byte, which is more efficient for our command based protocol

	for {
		line, err := reader.ReadString('\n') //read a line of input( when they press enter)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			conn.Close()
			return
		}

		outputs := strings.Fields(line) //split the line into fields based on whitespace. This allows us to parse commands like "SET key value 10s" into ["SET", "key", "value", "10s"]
		if len(outputs) == 0 {
			continue
		}
		command := strings.ToUpper(outputs[0])

		//handle the commands based on the first field. We can use a switch statement for this
		switch command {
		case "SET":
			var ttl time.Duration
			if len(outputs) < 3 {
				conn.Write([]byte("ERROR: Missing key or value\n"))
				continue
			}
			if len(outputs) >= 4 {
				ttl, _ = time.ParseDuration(outputs[3])
			}

			key := outputs[1]
			value := outputs[2]
			c.Set(key, value, ttl)
			conn.Write([]byte("OK\n"))

		case "GET":
			if len(outputs) < 2 {
				conn.Write([]byte("ERROR: Missing key\n"))
				continue
			}
			key := outputs[1]
			value, exists := c.Get(key)
			if !exists {
				conn.Write([]byte("NULL\n"))
			} else {
				conn.Write([]byte(value + "\n"))
			}
		case "DELETE":
			if len(outputs) < 2 {
				conn.Write([]byte("ERROR: Missing key\n"))
				continue
			}
			key := outputs[1]
			c.Delete(key)
			conn.Write([]byte("OK\n"))
		}
	}
}

func (c *Cache) Evict() {
	go func() { //need goroutine to run in background
		for {
			time.Sleep(time.Second)
			c.mu.Lock()
			for key, item := range c.items {
				if !item.expiration.IsZero() && time.Now().After(item.expiration) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		}
	}()
}
