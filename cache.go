package ttlcache

import (
	"sync"
	"time"
)

// Cache is a synchronised map of items that auto-expire once stale
type Cache struct {
	mutex   sync.RWMutex
	items   map[string]*Item
	counter uint64
}

// Set is a thread-safe way to add new items to the map
func (cache *Cache) Set(key string, data interface{}, ttl time.Duration, on_evict EvictCallback) {
	cache.mutex.Lock()
	item := &Item{data: data, ttl: ttl, on_evict: on_evict}
	item.touch()
	cache.items[key] = item
	cache.mutex.Unlock()
}

// Get is a thread-safe way to lookup items
// Every lookup, if touch set to true, touches the item, hence extending it's life
func (cache *Cache) Get(key string, touch bool) (data interface{}, found bool) {
	cache.mutex.Lock()
	item, exists := cache.items[key]
	if !exists || item.expired() {
		data = ""
		found = false
	} else {
		if touch {
			item.touch()
		}
		cache.counter++
		data = item.data
		found = true
	}
	cache.mutex.Unlock()
	return
}

func (cache *Cache) GetCounter() uint64 {
	return cache.counter
}

// Count returns the number of items in the cache
// (helpful for tracking memory leaks)
func (cache *Cache) Count() int {
	cache.mutex.RLock()
	count := len(cache.items)
	cache.mutex.RUnlock()
	return count
}

func (cache *Cache) cleanup() {
	cache.mutex.Lock()
	for key, item := range cache.items {
		if item.expired() {
			if item.on_evict != nil {
				item.on_evict(key, item.data)
			}
			delete(cache.items, key)
		}
	}
	cache.mutex.Unlock()
}

func (cache *Cache) startCleanupTimer() {
	duration := time.Second
	for i := range cache.items {
		if cache.items[i].ttl < time.Second {
			duration = cache.items[i].ttl
		}
	}
	ticker := time.Tick(duration)
	go (func() {
		for {
			select {
			case <-ticker:
				cache.cleanup()
			}
		}
	})()
}

func (cache *Cache) CleanAll() {
	cache.mutex.Lock()
	for key, _ := range cache.items {
		item := cache.items[key]
		if item.on_evict != nil {
			item.on_evict(key, item.data)
		}
		delete(cache.items, key)
	}
	cache.mutex.Unlock()
}

func (cache *Cache) Delete(key string) {
	cache.mutex.Lock()
	item := cache.items[key]
	if item.on_evict != nil {
		item.on_evict(key, item.data)
	}
	delete(cache.items, key)
	cache.mutex.Unlock()
}

// NewCache is a helper to create instance of the Cache struct
func NewCache() *Cache {
	cache := &Cache{
		items: map[string]*Item{},
	}
	cache.startCleanupTimer()
	return cache
}
