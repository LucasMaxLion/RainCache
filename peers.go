package rainCache

import "rainCache/raincachepb"

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	Get(in *raincachepb.Request, out *raincachepb.Response) error
}
