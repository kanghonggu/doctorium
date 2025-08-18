// app/encoding.go
package app

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// MakeEncodingConfig builds the EncodingConfig used in app.
// - InterfaceRegistry: protobuf interface registry
// - Marshaler: proto codec
// - Amino: legacy Amino codec
// - TxConfig: transaction config (sign modes)
func MakeEncodingConfig() EncodingConfig {

	interfaceRegistry := codectypes.NewInterfaceRegistry()

	authtypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)

	// Legacy Amino codec (필요시)
	amino := codec.NewLegacyAmino()
	ModuleBasics.RegisterLegacyAminoCodec(amino) // ★ 추가
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	// Protobuf codec
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	// TxConfig: protobuf-based signing modes
	txConfig := authtx.NewTxConfig(
		marshaler,
		authtx.DefaultSignModes, // 기본 sign mode 슬라이스
		// 추가 custom sign mode handler가 있다면 여기 적습니다…
	)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		Amino:             amino,
		TxConfig:          txConfig,
	}
}
