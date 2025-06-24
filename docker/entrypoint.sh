#!/bin/bash
set -e

APP=doctoriumd
CHAIN_ID=doctorium-test
MONIKER=validator01
KEY_NAME=validator01
KEYRING=file
HOME_DIR=/root/.doctoriumd
DENOM=stake
AMOUNT=100000000$DENOM
GAS_PRICE=0$DENOM
KEYRING_PASSPHRASE=12345678

echo "[entrypoint] Starting setup..."

# 1. 초기화
if [ ! -d "$HOME_DIR/config" ]; then
  echo "[entrypoint] Initializing chain..."
  $APP init $MONIKER --chain-id $CHAIN_ID --home $HOME_DIR
fi

# 2. 키 생성 (존재 안 하면)
if ! $APP keys show $KEY_NAME --keyring-backend $KEYRING --home $HOME_DIR &> /dev/null; then
  echo "[entrypoint] Creating key..."
  yes "$KEYRING_PASSPHRASE" | $APP keys add $KEY_NAME \
    --keyring-backend $KEYRING \
    --home $HOME_DIR \
    --output json
fi

# 3. 제네시스 계정 추가
ADDRESS=$($APP keys show $KEY_NAME --keyring-backend $KEYRING --home $HOME_DIR -a)
if ! grep -q "$ADDRESS" "$HOME_DIR/config/genesis.json"; then
  echo "[entrypoint] Adding genesis account..."
  $APP add-genesis-account "$ADDRESS" $AMOUNT --home $HOME_DIR
fi

# 4. gentx 생성
GENTX_PATH="$HOME_DIR/config/gentx/gentx-*.json"
if [ ! -f $GENTX_PATH ]; then
  echo "[entrypoint] Creating gentx..."
  yes "$KEYRING_PASSPHRASE" | $APP gentx $KEY_NAME $AMOUNT \
    --chain-id $CHAIN_ID \
    --home $HOME_DIR \
    --keyring-backend $KEYRING
fi

# 5. gentx 수집
echo "[entrypoint] Collecting gentxs..."
$APP collect-gentxs --home $HOME_DIR

# 6. 노드 시작
echo "[entrypoint] Starting node..."
exec $APP start --home $HOME_DIR --minimum-gas-prices=$GAS_PRICE
