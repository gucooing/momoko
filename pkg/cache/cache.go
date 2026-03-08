package cache

import (
	"sync"
	"time"
)

var (
	cacheTime = 5 * time.Minute
)

type (
	Key                 comparable
	Cache[K Key, V any] struct {
		dict sync.Map

		addLock sync.Mutex
		// 缓存时间
		cacheTime time.Duration
	}

	valueInterface struct {
		v          any
		activeTime time.Time
	}
)

func newValueInterface(v any) *valueInterface {
	return &valueInterface{
		v:          v,
		activeTime: time.Now(),
	}
}

func New[K Key, V any](cacheTime time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		dict:      sync.Map{},
		addLock:   sync.Mutex{},
		cacheTime: cacheTime,
	}
	go c.check()

	return c
}

func (c *Cache[K, V]) check() {
	ticker := time.NewTicker(time.Minute * 15)
	for range ticker.C {
		c.dict.Range(func(k, v any) bool {
			ojb := v.(*valueInterface)
			if ojb.activeTime.Add(c.cacheTime).Before(time.Now()) {
				c.dict.Delete(k)
			}
			return true
		})
	}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	ojb, ok := c.dict.Load(k)
	if !ok {
		var v V
		return v, false
	}
	ojb.(*valueInterface).activeTime = time.Now()

	return ojb.(*valueInterface).v.(V), ok
}

func (c *Cache[K, V]) GetByAdd(k K, fn func() (V, error)) (V, bool) {
	ojb, ok := c.dict.Load(k)
	if !ok {
		c.addLock.Lock()
		defer c.addLock.Unlock()
		if ojb, ok = c.dict.Load(k); ok {
			ojb.(*valueInterface).activeTime = time.Now()
			return ojb.(*valueInterface).v.(V), true
		}
		v, err := fn()
		if err != nil {
			return v, false
		}
		c.dict.Store(k, newValueInterface(v))
		return v, true
	}
	ojb.(*valueInterface).activeTime = time.Now()

	return ojb.(*valueInterface).v.(V), true
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	c.addLock.Lock()
	defer c.addLock.Unlock()
	ojb := newValueInterface(v)
	c.dict.Store(k, ojb)
	return true
}

func (c *Cache[K, V]) Del(k K) {
	c.dict.Delete(k)
}

func (c *Cache[K, V]) Clear() {
	c.dict = sync.Map{}
}
