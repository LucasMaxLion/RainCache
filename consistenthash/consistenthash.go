package consistenthash

import (
	"hash/crc32" // 导入crc32包用于生成哈希值
	"sort"       // 导入sort包用于排序操作
	"strconv"    // 导入strconv包用于字符串和数字之间的转换
)

// Hash 是一个函数类型，用于定义生成哈希值的函数
type Hash func(data []byte) uint32

// Map 结构体用于存储一致性哈希的相关信息
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 虚拟节点数
	keys     []int          // 存储哈希值的切片
	hashMap  map[int]string // 存储哈希值和对应的键的映射
}

// New 函数用于创建一个新的一致性哈希Map
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,                   // 设置传入的哈希函数
		replicas: replicas,             // 设置虚拟节点数
		hashMap:  make(map[int]string), // 初始化映射
	}
	if m.hash == nil { // 如果没有传入哈希函数，则使用crc32的IEEE实现
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 函数用于向一致性哈希中添加键
func (m *Map) Add(keys ...string) {
	for _, key := range keys { // 遍历传入的键
		for i := 0; i < m.replicas; i++ { // 对每个键生成多个哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) // 生成哈希值
			m.keys = append(m.keys, hash)                      // 将哈希值添加到keys切片中
			m.hashMap[hash] = key                              // 将哈希值和键添加到映射中
		}
	}
	sort.Ints(m.keys) // 对keys切片进行排序
}

// Get 函数用于根据键获取对应的值
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 { // 如果keys切片为空，则直接返回空字符串
		return ""
	}
	hash := int(m.hash([]byte(key))) // 计算键的哈希值

	// 使用二分查找找到第一个大于或等于hash的索引
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]] // 返回对应的键
}
