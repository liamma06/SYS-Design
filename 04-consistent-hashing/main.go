/*
Problem:
When you have servers storing data, you need to decided how to distribute the data across the servers.
Simple distribution is to use a hash function to map each data item to a server. For example, you could use the modulo operator to map each data item to a server based on its hash value.
However, this approach has a major drawback: when you add or remove a server, it can cause a large number of data items to be remapped to different servers. This can lead to significant downtime and performance issues.

Solution: Consistent Hashing

	Imagine a clock face (0–360). Hash your
	servers onto it. Hash your keys onto it too. A
	 key "belongs to" the nearest server
	clockwise.

	Remove a server: only the keys between it and
	the previous server need to move. Everything
	else stays put.
*/
package main

import (
	"fmt"
	"hash/crc32"
	"sort"
)

const vnodes = 10 // virtual nodes per server — more vnodes = more even distribution

type HashRing struct {
	positions []uint32          //list of values representing the positions of the servers on the hash ring
	nodes     map[uint32]string //mapping of positions to server identifiers (e.g. IP addresses)
	//map: key -> position on the ring, value -> server identifier
}

// point to the og hashring so we can modify it in the handler
func NewHashRing() *HashRing {
	return &HashRing{
		nodes: make(map[uint32]string),
	}
}

// convert key into a position on the hash ring using a hash function
func hash(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

// add node + sort the positions to maintain the order on the ring
// each server gets vnodes points on the ring for even distribution
func (r *HashRing) AddNode(name string) {
	for i := 0; i < vnodes; i++ {
		pos := hash(fmt.Sprintf("%s-%d", name, i)) // e.g. "server-A-0", "server-A-1"
		r.nodes[pos] = name
		r.positions = append(r.positions, pos)
	}
	//sort the positions to maintain the order on the ring
	sort.Slice(r.positions, func(i, j int) bool {
		return r.positions[i] < r.positions[j]
	})
}

// remove all vnodes for this server from the ring
func (r *HashRing) RemoveNode(name string) {
	for i := 0; i < vnodes; i++ {
		pos := hash(fmt.Sprintf("%s-%d", name, i))
		delete(r.nodes, pos)
		for j, p := range r.positions {
			if p == pos {
				r.positions = append(r.positions[:j], r.positions[j+1:]...)
				break
			}
		}
	}
}

func (r *HashRing) GetNode(key string) string {
	if len(r.positions) == 0 {
		return ""
	}

	pos := hash(key)

	for _, ringPos := range r.positions {
		if pos <= ringPos {
			return r.nodes[ringPos]
		}
	}
	return r.nodes[r.positions[0]]
}

func main() {
	ring := NewHashRing()
	ring.AddNode("server-A")
	ring.AddNode("server-B")
	ring.AddNode("server-C")

	// show distribution across 1000 keys — should be roughly even
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		node := ring.GetNode(fmt.Sprintf("key:%d", i))
		counts[node]++
	}
	fmt.Println("Distribution:", counts)

	// show that removing a node only moves some keys
	fmt.Println("\n--- Before removal ---")
	keys := []string{"user:1", "user:2", "user:3", "order:99"}
	for _, key := range keys {
		fmt.Printf("%s -> %s\n", key, ring.GetNode(key))
	}

	ring.RemoveNode("server-B")

	fmt.Println("--- After removing server-B ---")
	for _, key := range keys {
		fmt.Printf("%s -> %s\n", key, ring.GetNode(key))
	}
}
