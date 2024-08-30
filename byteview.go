package rainCache

/*
前面实现的LFU返回是value占了多少byte
*/

type ByteView struct {
	b []byte
}

/*
返回byteview的大小，也就是有多少byte
*/

func (v ByteView) Len() int {
	return len(v.b)
}

/*
深度copy
*/

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

/*
绑定一个方法，但是TMD 这两个方法作用不应该是一样的吗？可能是为了在后面使用的时候，方便理解
*/

func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}
