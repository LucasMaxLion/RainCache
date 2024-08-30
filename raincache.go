package rainCache

import (
	"fmt"
	"log"
	"rainCache/raincachepb"
	"rainCache/singleflight"
	"sync"
)

/*
                            是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

/*
mu是全局读写写
groups 缓存的大表
*/

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

type Getter interface {
	Get(key string) ([]byte, error)
}
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

/*
name 缓存唯一的名称
getter 缓存未命中时获取源数据的回调(callback)？
maincache 就是我们实现的缓存
*/

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	//使用这个确保每个key只被调用一次
	loader *singleflight.Group
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// NewGroup 新建一个group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

/*
这里使用了读锁
*/

// GetGroup 查找对应的group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

/*
get方法，获取缓存对应的内容，
*/

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[rainCache] hit")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &raincachepb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &raincachepb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

func (g *Group) load(key string) (vale ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[RainCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err != nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
