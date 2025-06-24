package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure MsgUploadFile implements the sdk.Msg interface
var _ sdk.Msg = &MsgUploadFile{}

// Route implements sdk.Msg
func (msg *MsgUploadFile) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgUploadFile) Type() string {
	return "UploadFile"
}

// ValidateBasic implements sdk.Msg
func (msg *MsgUploadFile) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.FileHash == "" {
		return fmt.Errorf("file hash cannot be empty")
	}
	return nil
}

// GetSignBytes implements sdk.Msg
func (msg *MsgUploadFile) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// GetSigners implements sdk.Msg
func (msg *MsgUploadFile) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}
