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
	"fmt"
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
	c := &Cache{items: make(map[string]item)}
	c.Evict()

	// key with a 2s TTL
	c.Set("hello", "world", 2*time.Second)
	if val, ok := c.Get("hello"); ok {
		fmt.Println("found:", val)
	}

	// key with no TTL — lives forever
	c.Set("permanent", "stays", 0)

	time.Sleep(3 * time.Second)

	if _, ok := c.Get("hello"); !ok {
		fmt.Println("hello expired, evicted by background goroutine")
	}
	if val, ok := c.Get("permanent"); ok {
		fmt.Println("permanent still here:", val)
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
