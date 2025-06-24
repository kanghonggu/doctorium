package types

const (
	ModuleName   = "filehash" // 모듈 이름
	RouterKey    = ModuleName // Msg 라우팅 시 사용
	QuerierRoute = ModuleName // Querier 라우팅 시 사용
	StoreKey     = ModuleName // KVStore key
)

var (
	FileKeyPrefix = []byte{0x01}
)
