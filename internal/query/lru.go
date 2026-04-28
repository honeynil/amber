package query

import (
	"container/list"
	"sync"
)

type indexLRU[V any] struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type lruEntry[V any] struct {
	key string
	val V
}

func newIndexLRU[V any](capacity int) *indexLRU[V] {
	if capacity < 1 {
		capacity = 1
	}
	return &indexLRU[V]{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

func (c *indexLRU[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*lruEntry[V]).val, true
}

func (c *indexLRU[V]) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.Remove(el)
		delete(c.items, key)
	}
}

func (c *indexLRU[V]) put(key string, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		el.Value.(*lruEntry[V]).val = val
		c.order.MoveToFront(el)
		return
	}
	entry := &lruEntry[V]{key: key, val: val}
	el := c.order.PushFront(entry)
	c.items[key] = el
	if c.order.Len() > c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*lruEntry[V]).key)
		}
	}
}
