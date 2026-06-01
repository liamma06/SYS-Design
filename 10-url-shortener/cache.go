/*
simple in-memory cache implementation for url shortener service
so retrieve long url from cache before hitting the store/database
*/
package main

import (
	"sync"
	"time"
)

type Item struct {
	url        string
	expiration time.Time
}

type Cache struct {
	items map[string]Item //string(short code) -> Item
	rw    sync.RWMutex
}

// constructor
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]Item), //initialize empty map
	}
}

// create new entry in cache with expiration time
func (c *Cache) Set(code string, url string, duration time.Duration) {
	c.rw.Lock() //lock for writing
	defer c.rw.Unlock()

	c.items[code] = Item{
		url:        url,
		expiration: time.Now().Add(duration), //set expiration time
	}

}

func (c *Cache) Get(code string) (string, bool) {
	c.rw.RLock() //lock for reading
	defer c.rw.RUnlock()

	item, exists := c.items[code]
	if !exists {
		return "", false
	}
	if item.expiration.IsZero() {
		return item.url, true //no expiration time set, return url
	}
	return item.url, time.Now().Before(item.expiration) //return url and check if not expired
}

func (c *Cache) Evict() {
	go func() {
		for {
			time.Sleep(time.Minute)
			c.rw.Lock() //lock for writing
			for code := range c.items {
				if time.Now().After(c.items[code].expiration) { //check if expired
					if !c.items[code].expiration.IsZero() { //if expiration time is set, delete item
						delete(c.items, code) //remove expired item
					}
				}
			}
			c.rw.Unlock()
		}
	}()

}
