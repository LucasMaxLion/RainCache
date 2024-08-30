package rainCache

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"rainCache/consistenthash"
	"rainCache/raincachepb"
	"strings"
	"sync"
)

const defaultBasePath = "/_raincache/"
const defaultReplicas = 50

/*
self 用来记录自己的地址，包括主机名/IP 和端口
basePath，作为节点间通讯地址的前缀

*/

// HTTPPool 作为承载节点间 HTTP 通信的核心数据结构
type HTTPPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 用来打印日柱的
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 定义一个HTTPPool类型的ServeHTTP方法，该方法实现了http.Handler接口。
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查请求的URL路径是否以HTTPPool的basePath开始。
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		// 如果不是，则引发panic错误。
		panic("HTTPPool serving unexpected path:" + r.URL.Path)
	}
	// 记录请求的方法和路径。
	p.Log("%s %s", r.Method, r.URL.Path)

	// 从basePath之后的部分开始，将URL路径分割成两部分。
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	// 检查分割后的结果是否正好是两部分。
	if len(parts) != 2 {
		// 如果不是，则返回400 Bad Request错误。
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// 第一部分是组名。
	groupName := parts[0]
	// 第二部分是键。
	key := parts[1]

	// 根据组名获取对应的组。
	group := GetGroup(groupName)
	// 如果没有找到组，则返回404 Not Found错误。
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	// 从组中获取键对应的视图。
	view, err := group.Get(key)
	// 如果获取失败，则返回500 Internal Server Error错误。
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 将视图序列化为字节流。
	body, err := proto.Marshal(&raincachepb.Response{Value: view.ByteSlice()})
	// 如果序列化失败，则返回500 Internal Server Error错误。
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置响应的Content-Type为application/octet-stream。
	w.Header().Set("Content-Type", "application/octet-stream")
	// 将序列化后的字节流写入响应体。
	w.Write(body)
}

type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(in *raincachepb.Request, out *raincachepb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

func (p *HTTPPool) PickPeer(key string) (peer PeerGetter, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
