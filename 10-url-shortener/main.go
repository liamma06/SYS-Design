/*
1. shortner -> URL shortener
2. store -> database
3. cache -> in-memory cache
4. ratelimiter -> limit number of requests per user/IP
5. logger -> log requests and errors
6. message queue -> for analyzing traffic patterns and generating reports
7. main -> entry point, set up server and routes
*/
package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	//initalize
	store := NewStore()
	cache := NewCache()
	ratelimiter := NewRateLimiter()
	logger := NewLogger("url-shortener")
	queue := NewQueue()
	cache.Evict()                       //start cache eviction in background
	queue.Subscribe(func(msg Message) { //subscribe to message queue
		fmt.Println("event:", string(msg.Payload))
	})

	//HTTP handler for shortening URLs
	http.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		//post checker
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		//rate limit check
		userID := r.RemoteAddr //use IP as user ID
		if !ratelimiter.IsAllowed(userID) {
			http.Error(w, "Rate limit exceeded", 429)
			return
		}

		//parse request body for original URL
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		url := string(body) //grab url from body

		//generate shortner code
		code := GenerateCode()

		//store in DB
		store.Save(code, url)

		//log request
		logger.Log(INFO, "url shortened: "+url)

		//publissh event to queue for analysis
		queue.Publish(Message{
			ID:        code,
			Payload:   []byte("url.created"),
			Timestamp: time.Now().Unix(),
		})

		fmt.Fprint(w, code) //respond with short code
	})

	http.HandleFunc("/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Path[1:] //extract code from path

		//check rate limit
		userID := r.RemoteAddr
		if !ratelimiter.IsAllowed(userID) {
			http.Error(w, "Rate limit exceeded", 429)
			return
		}

		//check cache first
		url, found := cache.Get(code)
		if found {
			http.Redirect(w, r, url, http.StatusFound)
			logger.Log(INFO, "url accessed: "+code)
			return
		}

		//not in cache, check DB
		url, found = store.Get(code)
		if found {
			cache.Set(code, url, 0) //cache it for future requests ( no expiration)
			http.Redirect(w, r, url, http.StatusFound)
			logger.Log(INFO, "url accessed: "+code)
			return
		}

		http.Error(w, "URL not found", http.StatusNotFound)
		return

	})

	http.ListenAndServe(":8080", nil) //start server on port 8080
}
