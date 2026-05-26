package main

//from 04-consistent-hashing/main.go

import (
	"fmt"
	"hash/crc32"
	"sort"
)

const vnodes = 10

type HashRing struct {
	positions []uint32
	nodes     map[uint32]string
}

func NewHashRing() *HashRing {
	return &HashRing{
		nodes: make(map[uint32]string),
	}
}

func hash(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

func (r *HashRing) AddNode(name string) {
	for i := 0; i < vnodes; i++ {
		pos := hash(fmt.Sprintf("%s-%d", name, i))
		r.nodes[pos] = name
		r.positions = append(r.positions, pos)
	}
	sort.Slice(r.positions, func(i, j int) bool {
		return r.positions[i] < r.positions[j]
	})
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
