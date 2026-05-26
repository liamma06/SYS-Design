/*
load balancer distirbutes incoming requests and decides which backend server should handle it

*/

package main

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// server struct
// URL: the address of the backend server
// alive: whether the server is healthy and can receive traffic
// mu: a mutex to protect concurrent access to the alive status
type Backend struct {
	URL         *url.URL
	alive       bool
	mu          sync.RWMutex
	activeConns int64 //number of active connections to this backend
}

type BackendPool struct {
	backends []*Backend
	current  uint64 //index of curret backend(used for increment safely across goroutines)
}

func main() {
	//create backend pool and add some backends
	pool := &BackendPool{}
	pool.backends = append(pool.backends, &Backend{URL: &url.URL{Host: "localhost:8081"}, alive: true})
	pool.backends = append(pool.backends, &Backend{URL: &url.URL{Host: "localhost:8082"}, alive: true})
	go pool.healthCheck()

	http.HandleFunc("/", handleRequest(pool)) //prebuilt handler function that takes the pool as an argument and returns a handler function that can be used with http.HandleFunc
	http.ListenAndServe(":8080", nil)

}

func handleRequest(pool *BackendPool) http.HandlerFunc {
	//return a handler function that can be used with http.HandleFunc
	//this confused me as to why return a func but basically "when someone comes run this func and we'll give to you."
	return func(w http.ResponseWriter, r *http.Request) {
		backend := pool.lowestConns()
		if backend == nil {
			http.Error(w, "no backends available", http.StatusServiceUnavailable)
			return
		}

		atomic.AddInt64(&backend.activeConns, 1)
		defer atomic.AddInt64(&backend.activeConns, -1)

		//send request to backend using reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(backend.URL)
		proxy.ServeHTTP(w, r)
	}
}

func (pool *BackendPool) lowestConns() *Backend {
	var lowest *Backend
	for _, b := range pool.backends {
		if (lowest == nil || b.ActiveConns() < lowest.ActiveConns()) && b.IsAlive() {
			lowest = b
		}
	}
	return lowest
}

func (b *Backend) ActiveConns() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.activeConns
}

func (b *Backend) IsAlive() bool {
	//note: uses R lock because it is just reading the alive status and this allow multiple go routines to read at the same time
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.alive
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock() //ensure mutex is unlocked even if panic occurs,  unlock whenever the function is complete
	b.alive = alive
}

func (pool *BackendPool) healthCheck() {
	for {
		time.Sleep(10 * time.Second) //check every 10 seconds

		for _, b := range pool.backends {
			check := checkBackend(b)
			b.SetAlive(check)
		}
	}
}

func checkBackend(b *Backend) bool {
	//try to connect to the backend with a timeout
	con, err := net.DialTimeout("tcp", b.URL.Host, 2*time.Second)
	if err != nil {
		return false
	}
	con.Close() //close connection if success.
	return true
}
