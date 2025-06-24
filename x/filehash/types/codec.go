package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ModuleCdc is the global Legacy Amino codec.  Remove if you're not using Amino.
var ModuleCdc = codec.NewLegacyAmino()

// RegisterLegacyAminoCodec registers concrete types on the Amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUploadFile{}, "doctorium/filehash/MsgUploadFile", nil)
}

// RegisterInterfaces registers module message and service interfaces
// with the protobuf InterfaceRegistry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUploadFile{},
	)
	// 서비스 인터페이스도 등록하려면:
	// types.RegisterMsgServer(registry, /* your server impl */)
}
