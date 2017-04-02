package ttlcache

import (
	"sync"
	"time"
)

// Item represents a record in the cache map
type EvictCallback func(key string, value interface{})

type Item struct {
	sync.RWMutex
	data     interface{}
	expires  *time.Time
	ttl      time.Duration
	on_evict EvictCallback
}

func (item *Item) touch() {
	item.Lock()
	expiration := time.Now().Add(item.ttl)
	item.expires = &expiration
	item.Unlock()
}

func (item *Item) expired() bool {
	var value bool
	item.RLock()
	if item.expires == nil {
		value = true
	} else {
		value = item.expires.Before(time.Now())
	}
	item.RUnlock()
	return value
}
