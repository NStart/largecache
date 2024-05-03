package largecache

type hashSub uint64

func (stub hashSub) Sum64(_ string) uint64 {
	return uint64(stub)
}
