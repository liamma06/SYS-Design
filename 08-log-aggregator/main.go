/*
Log aggregator is a program that collects logs from multiplie sources and aggregates them into a single stream. This can be useful for monitoring and debugging purposes, as it allows you to see all of your logs in one place.
*/
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type Level string //custom type for log levels

// log levels for categorizing log messages, allowing for filtering and prioritization of logs based on their severity.
const (
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

// if converted to json it would take those fields
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
	Service   string    `json:"service"` //service that generated the log entry
}

// a ring buffer so it replaces old log entries with new ones
type Store struct {
	mu      sync.RWMutex //mutex allow read but not concurrent writes
	entries []LogEntry
	head    int //index of oldest log entry
	count   int
	cap     int
}

func NewStore(capacity int) *Store {
	return &Store{
		entries: make([]LogEntry, capacity), //empty entries with fixed cap
		cap:     capacity,
		head:    0,
		count:   0,
	}
}

func (s *Store) Write(e LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[s.head] = e
	s.head = (s.head + 1) % s.cap //move head to next position, wrap around if it reaches the end

	//once full it will start overwriting old log entries, so count will not exceed capacity
	if s.count < s.cap {
		s.count++
	}

}

// return all logs sorted (2 cases depending if wrap happened)
func (s *Store) All() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]LogEntry, s.count)

	if s.count < s.cap {
		copy(result, s.entries[:s.count]) //return logs with no wrapping
		return result
	}
	//logs after head
	n := copy(result, s.entries[s.head:]) //everything after head
	copy(result[n:], s.entries[:s.head])  //everything before head
	return result
}

func main() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}

	store := NewStore(10)              //store with cap of 10
	go startHTTPServer(":8080", store) //start http server in a separate goroutine

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn, store)
	}
}

func handleConnection(conn net.Conn, store *Store) {
	defer conn.Close() //close connection when done

	decoder := json.NewDecoder(conn)

	for {
		var entry LogEntry //entry has to follow the struct

		err := decoder.Decode(&entry) //decode json log entry from connection
		if err != nil {
			fmt.Println("Error decoding log entry:", err)
			return
		}
		store.Write(entry) //write log entry to store
		fmt.Printf("Received log: %s [%s] %s\n", entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)
	}
}

func startHTTPServer(addr string, store *Store) {
	//http server to serve logs via an API endpoint
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		allLogs := store.All() //get all logs from store
		query := r.URL.Query()

		levelFilter := query.Get("level") //get log level from query
		if levelFilter != "" {
			filtered := []LogEntry{}
			//loop through to filter
			for _, log := range allLogs {
				if string(log.Level) == levelFilter { //filter logs by level if specified
					filtered = append(filtered, log)
				}
			}
			allLogs = filtered
		}

		serviceFilter := query.Get("service") //get service name from query
		if serviceFilter != "" {
			filtered := []LogEntry{}
			for _, log := range allLogs {
				if log.Service == serviceFilter { //filter logs by service if specified
					filtered = append(filtered, log)
				}
			}
			allLogs = filtered
		}

		sinceFilter := query.Get("since") // e.g. "60s", "5m", "1h"
		if sinceFilter != "" {
			duration, err := time.ParseDuration(sinceFilter)
			if err == nil {
				cutoff := time.Now().Add(-duration) //calculate cutoff time based on duration
				filtered := []LogEntry{}
				for _, log := range allLogs {
					if log.Timestamp.After(cutoff) {
						filtered = append(filtered, log)
					}
				}
				allLogs = filtered
			}
		}
		//set response header and encode logs as json in response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allLogs)
	})

	http.ListenAndServe(addr, nil) //start http server on specified address
}
