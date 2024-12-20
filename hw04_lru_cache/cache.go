package hw04lrucache

import (
	"sync"
)

type Key string

type Cache interface {
	Set(key Key, value interface{}) bool
	Get(key Key) (interface{}, bool)
	Clear()
}

type lruCache struct {
	capacity int
	queue    List
	items    map[Key]*ListItem
	vals     map[*ListItem]Key
	mutex    sync.RWMutex
}

func NewCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[Key]*ListItem, capacity),
		vals:     make(map[*ListItem]Key, capacity),
	}
}

func (lru *lruCache) Set(key Key, value interface{}) bool {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	node := lru.items[key]
	if node != nil {
		node.Value = value
		lru.queue.MoveToFront(node)
		return true
	}
	if lru.queue.Len() == lru.capacity {
		lastNode := lru.queue.Back()
		nodeKey := lru.vals[lastNode]
		delete(lru.items, nodeKey)
		delete(lru.vals, lastNode)
		lru.queue.Remove(lastNode)
	}

	node = lru.queue.PushFront(value)
	lru.items[key] = node
	lru.vals[node] = key

	return false
}

func (lru *lruCache) Get(key Key) (interface{}, bool) {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()

	node := lru.items[key]
	if node == nil {
		return nil, false
	}
	lru.queue.MoveToFront(node)
	return node.Value, true
}

func (lru *lruCache) Clear() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	lru.items = make(map[Key]*ListItem, lru.capacity)
	lru.vals = make(map[*ListItem]Key, lru.capacity)
	lru.queue = NewList()
}
