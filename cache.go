package rainCache

import (
	"rainCache/lfu"
	"sync"
)

/*
三个字段，分别是锁，一个指向lfu的指针，一个cacheBytes，意思是lfu支持的最大内存
*/

type cache struct {
	mu         sync.Mutex
	lfu        *lfu.Cache
	cacheBytes int64
}

/*
新增，使用锁保证并发安全
*/
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lfu == nil {
		c.lfu = lfu.New(c.cacheBytes, nil)
	}
	c.lfu.Add(key, value)
}

/*
get方法，从lfu中获取对应的值
*/
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lfu == nil {
		return
	}
	if v, ok := c.lfu.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}

/*
总结：
这个cache是对lfu的封装，
主要做了一件事情，就是保证lfu可以并发
*/
