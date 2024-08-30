package lfu

import (
	"container/list"
	"math"
)

// Object 一个key value 元素
type Object struct {
	key   string
	value Value
	freq  int64 //频率
}

func InitObject(k string, v Value, freq int64) *Object {
	return &Object{
		key:   k,
		value: v,
		freq:  freq,
	}
}

type Cache struct {
	len, cap int64 //长度与容量
	minFreq  int64 //目前c缓存中，操作频次最小的元素
	// key: 元素的key
	// val: 元素的节点
	objectMap map[string]*list.Element
	// key: nodeMap中所有元素出现的可能频次
	// val: NodeList 频次相同头部的元素 操作时间离现在最近
	freqMap   map[int64]*list.List
	OnEvicted func(key string, value Value)
}

type Value interface {
	Len() int
}

func New(capacity int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		len:       0,
		cap:       capacity,
		minFreq:   math.MaxInt,
		objectMap: make(map[string]*list.Element),
		freqMap:   make(map[int64]*list.List),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	// 存在返回val
	if e, ok := c.objectMap[key]; ok {
		// e元素操作频率+1
		// 调整freqMap
		c.increaseFreq(e)
		ob := e.Value.(*Object)
		return ob.value, true
	}
	// 不存在返回
	return
}

func (c *Cache) Add(key string, value Value) {
	if e, ok := c.objectMap[key]; ok {
		// key已经存在了，更新val
		ob := e.Value.(*Object)
		ob.value = value
		c.increaseFreq(e)
	} else {
		// key不存在
		ob := InitObject(key, value, 1)
		// 1.如果满了，淘汰
		if c.len == c.cap {
			c.eliminate()
			c.len--
		}
		// 2.插入freqMap与objectMap
		c.insertMap(ob)
		// 3.调整minFreq, 因为插入了一个新的，所以一定是1
		c.minFreq = 1
		// 4.现有数量++
		c.len++
	}
	return
}

// increaseFreq e元素操作频次+1，调整其所在的list
func (c *Cache) increaseFreq(e *list.Element) {
	ob := e.Value.(*Object)
	// 1.先从低频次移除
	oldList := c.freqMap[ob.freq]
	oldList.Remove(e)
	// 2.调整minFreq
	// 如果移除的就是最小频次的最后一个节点
	if c.minFreq == ob.freq && oldList.Len() == 0 {
		c.minFreq++
	}
	// 3.再添加到高频次list
	ob.freq++
	c.insertMap(ob)
}

func (c *Cache) insertMap(ob *Object) {
	// 1.添加到freqMap中
	l, ok := c.freqMap[ob.freq]
	if !ok {
		l = list.New()
		c.freqMap[ob.freq] = l
	}
	e := l.PushFront(ob)
	// 2.添加/更新objectMap中的e元素为进入了新list中的e
	c.objectMap[ob.key] = e
}

// eliminate 淘汰c中频次最小节点,如果频次相同淘汰距离现在时间最长的
func (c *Cache) eliminate() {
	l := c.freqMap[c.minFreq]
	e := l.Back()
	ob := e.Value.(*Object)

	l.Remove(e)
	delete(c.objectMap, ob.key)
}
