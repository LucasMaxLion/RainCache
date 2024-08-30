package singleflight

import (
	"sync"
)

// 定义一个call结构体，用于存储并发执行的结果和错误。
type call struct {
	wg  sync.WaitGroup // 用于等待goroutine完成
	val interface{}    // 存储执行结果
	err error          // 存储执行过程中的错误
}

// Group 定义一个Group结构体，用于管理并发执行的调用。
type Group struct {
	mu sync.Mutex       // 用于同步访问map
	m  map[string]*call // 存储key对应的call对象
}

// Do Do方法接受一个键和一个函数，执行函数并返回结果。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock() // 锁定互斥锁，确保线程安全
	if g.m == nil {
		g.m = make(map[string]*call) // 如果map不存在，则创建一个
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock() // 如果map中已存在对应的call对象，则解锁并等待goroutine完成
		c.wg.Wait()
		return c.val, c.err // 返回存储的结果和错误
	}
	c := new(call) // 创建一个新的call对象
	c.wg.Add(1)    // 增加WaitGroup的计数
	g.m[key] = c   // 将call对象存储到map中
	g.mu.Unlock()  // 解锁

	c.val, c.err = fn() // 执行传入的函数，并存储结果和错误
	c.wg.Done()         // 减少WaitGroup的计数，通知等待的goroutine

	g.mu.Lock()      // 再次锁定互斥锁
	delete(g.m, key) // 从map中删除对应的call对象
	g.mu.Unlock()    // 解锁

	return c.val, c.err // 返回存储的结果和错误
}
