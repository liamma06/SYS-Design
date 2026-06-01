/*
mapping of short code to long url
*/
package main

import (
	"sync"
)

type Store struct {
	urlMap map[string]string //string(short code) -> string(url)
	rw     sync.RWMutex
}

// constructor
func NewStore() *Store {
	return &Store{
		urlMap: make(map[string]string), //initialize empty map
	}
}

func (s *Store) Save(code string, url string) {
	s.rw.Lock() //lock for writing
	defer s.rw.Unlock()

	s.urlMap[code] = url
}

func (s *Store) Get(code string) (string, bool) {
	s.rw.RLock() //lock for reading
	defer s.rw.RUnlock()

	url, exists := s.urlMap[code]
	return url, exists
}
