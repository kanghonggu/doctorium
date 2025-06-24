## doctorium
> Creator: hgkang
> Date: 2025/06/24

## Description 
* doctorium에 대한 설명을 작성하세요

# 1. 초기화
./build/doctoriumd init validator01 --chain-id doctorium-test --home ~/.doctoriumd

# 2. 키 생성 (secp256k1 명시)
./build/doctoriumd keys add validator01 --keyring-backend file --algo secp256k1 --home ~/.doctoriumd

# 3. 제네시스 계정 추가 (많은 사람들이 이걸 빼먹음)
ADDRESS=$(./build/doctoriumd keys show validator01 -a --keyring-backend file --home ~/.doctoriumd)
./build/doctoriumd add-genesis-account "$ADDRESS" 100000000stake --home ~/.doctoriumd

# 4. gentx 시 pubkey 명시 (옵션)
PUBKEY=$(./build/doctoriumd keys show validator01 --keyring-backend file --home ~/.doctoriumd --output json | jq -r '.pubkey')
./build/doctoriumd gentx validator01 100000000stake \
--chain-id doctorium-test \
--keyring-backend file \
--home ~/.doctoriumd \
--pubkey "$PUBKEY"
