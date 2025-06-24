package types

import "fmt"

// ValidateGenesis checks that the genesis state is valid.
// 여기서 필요한 검증 로직(중복 해시 없음 등)을 넣으시면 됩니다.
func ValidateGenesis(data *GenesisState) error {
	// 예시: FileHash가 중복되지 않는지 확인
	seen := make(map[string]struct{})
	for _, f := range data.Files {
		if _, exists := seen[f.FileHash]; exists {
			return fmt.Errorf("duplicate file hash in genesis: %s", f.FileHash)
		}
		seen[f.FileHash] = struct{}{}
	}
	return nil
}
