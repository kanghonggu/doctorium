#!/usr/bin/env bash
set -euo pipefail

log(){ >&2 echo "[$(date '+%F %T')] [entrypoint] $*"; }

APP="${APP:-doctoriumd}"
CHAIN_ID="${CHAIN_ID:-doctorium-test}"
MONIKER="${MONIKER:-validator01}"
KEY_NAME="${KEY_NAME:-validator01}"
KEYRING="${KEYRING:-file}"                  # file | test
KEYRING_PASSPHRASE="${KEYRING_PASSPHRASE:-12345678}"

HOME_DIR="${HOME_DIR:-/root/.doctoriumd}"
DENOM="${DENOM:-stake}"
GENESIS_COINS="${GENESIS_COINS:-100000000000${DENOM}}"
SELF_DELEGATE="${SELF_DELEGATE:-100000000${DENOM}}"
MIN_GAS_PRICE="${MIN_GAS_PRICE:-0.025${DENOM}}"

APP_TOML="$HOME_DIR/config/app.toml"
CONFIG_TOML="$HOME_DIR/config/config.toml"

# 디버그용: HOLD=1이면 쉘로 들어갈 수 있게 대기
if [[ "${HOLD:-0}" == "1" ]]; then
  log "HOLD=1; waiting for debug (tail -f /dev/null)"
  tail -f /dev/null
fi

is_valid_addr(){ [[ "${1:-}" =~ ^cosmos1[0-9a-z]{38}$ ]]; }

ensure_dirs(){
  mkdir -p "$HOME_DIR" "$HOME_DIR/config" "$HOME_DIR/keyring-file" "$HOME_DIR/config/gentx"
  chmod 700 "$HOME_DIR/keyring-file" || true
}

init_chain(){
  if [[ ! -f "$HOME_DIR/config/genesis.json" ]]; then
    log "Initializing chain..."
    "$APP" init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"
  fi
}

create_key_file(){
  log "Creating key (file backend)…"
  : >/tmp/key.stdout; : >/tmp/key.stderr; : >/tmp/key.json

  # 1) 실행: stdout / stderr 분리 저장
  printf "%s\n%s\n" "$KEYRING_PASSPHRASE" "$KEYRING_PASSPHRASE" | \
    "$APP" keys add "$KEY_NAME" --algo secp256k1 \
      --keyring-backend file --keyring-dir "$HOME_DIR" --home "$HOME_DIR" \
      --output json 1>/tmp/key.stdout 2>/tmp/key.stderr || true

  # 2) 어떤 스트림에 JSON이 있는지 판별
  if jq -e . >/dev/null 2>&1 </tmp/key.stdout; then
    cp /tmp/key.stdout /tmp/key.json
  elif jq -e . >/devnull 2>&1 </tmp/key.stderr; then
    cp /tmp/key.stderr /tmp/key.json
  else
    # 로그가 앞뒤에 섞였을 가능성 → 합쳐서 {..} 부분만 추출 시도
    cat /tmp/key.stdout /tmp/key.stderr > /tmp/key.raw
    awk 'BEGIN{p=0} /\{/ {p=1} {if(p)print} /\}/ {if(p){exit}}' /tmp/key.raw > /tmp/key.json || true
  fi

  # 최종 유효성
  if ! jq -e . >/dev/null 2>&1 </tmp/key.json; then
    log "keys add (file) produced no valid JSON"
    >&2 echo "STDOUT:"; >&2 sed -n '1,200p' /tmp/key.stdout
    >&2 echo "STDERR:"; >&2 sed -n '1,200p' /tmp/key.stderr
    exit 1
  fi
}

create_key_test(){
  log "Creating key (test backend)…"
  : >/tmp/key.stdout; : >/tmp/key.stderr; : >/tmp/key.json

  "$APP" keys add "$KEY_NAME" --algo secp256k1 \
    --keyring-backend test --keyring-dir "$HOME_DIR" --home "$HOME_DIR" \
    --output json 1>/tmp/key.stdout 2>/tmp/key.stderr || true

  if jq -e . >/dev/null 2>&1 </tmp/key.stdout; then
    cp /tmp/key.stdout /tmp/key.json
  elif jq -e . >/dev/null 2>&1 </tmp/key.stderr; then
    cp /tmp/key.stderr /tmp/key.json
  else
    cat /tmp/key.stdout /tmp/key.stderr > /tmp/key.raw
    awk 'BEGIN{p=0} /\{/ {p=1} {if(p)print} /\}/ {if(p){exit}}' /tmp/key.raw > /tmp/key.json || true
  fi

  if ! jq -e . >/dev/null 2>&1 </tmp/key.json; then
    log "keys add (test) produced no valid JSON"
    >&2 echo "STDOUT:"; >&2 sed -n '1,200p' /tmp/key.stdout
    >&2 echo "STDERR:"; >&2 sed -n '1,200p' /tmp/key.stderr
    exit 1
  fi
}


keys_show_addr(){
  local backend="$1"
  if [[ "$backend" == "file" ]]; then
    printf "%s" "$KEYRING_PASSPHRASE" | \
      "$APP" keys show "$KEY_NAME" -a \
        --keyring-backend file --keyring-dir "$HOME_DIR" --home "$HOME_DIR"
  else
    "$APP" keys show "$KEY_NAME" -a \
      --keyring-backend test --keyring-dir "$HOME_DIR" --home "$HOME_DIR"
  fi
}

get_address_or_fail(){
  local addr=""
  [[ -f /tmp/key.json ]] && addr="$(jq -r '.address // empty' /tmp/key.json 2>/dev/null || true)"
  if ! is_valid_addr "$addr"; then
    : > /tmp/keys_show.err
    addr="$(keys_show_addr "$1" 2>/tmp/keys_show.err || true)"
  fi
  if ! is_valid_addr "$addr"; then
    log "Failed to obtain address. Diagnostics:"
    >&2 echo "KEYRING=$KEYRING"
    >&2 echo "keys_add.err:"; >&2 cat /tmp/keys_add.err 2>/dev/null || true
    >&2 echo "keys_show.err:"; >&2 cat /tmp/keys_show.err 2>/dev/null || true
    >&2 echo "key.json:"; >&2 head -200 /tmp/key.json 2>/dev/null || true
    return 1
  fi
  echo "$addr"
}

recover_corrupted_file_ring(){
  log "Detected corrupted keyring-file. Recreating…"
  rm -rf "$HOME_DIR/keyring-file"
  mkdir -p "$HOME_DIR/keyring-file"
  chmod 700 "$HOME_DIR/keyring-file" || true
}

gentx_with_backend(){
  local backend="$1"
  if [[ "$backend" == "file" ]]; then
    printf "%s" "$KEYRING_PASSPHRASE" | \
      "$APP" gentx "$KEY_NAME" "$SELF_DELEGATE" \
        --chain-id "$CHAIN_ID" --keyring-backend file --keyring-dir "$HOME_DIR" --home "$HOME_DIR"
  else
    "$APP" gentx "$KEY_NAME" "$SELF_DELEGATE" \
      --chain-id "$CHAIN_ID" --keyring-backend test --keyring-dir "$HOME_DIR" --home "$HOME_DIR"
  fi
}

configure_endpoints(){
  if [[ -f "$APP_TOML" ]]; then
    sed -i 's/^\s*enable = false/enable = true/' "$APP_TOML" || true
    sed -i 's|^\s*address = "tcp://127\.0\.0\.1:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML" || true
    sed -i 's/^\s*swagger = false/swagger = true/' "$APP_TOML" || true
    sed -i 's/^\s*enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML" || true
    sed -i 's|^\s*address = "127\.0\.0\.1:9090"|address = "0.0.0.0:9090"|' "$APP_TOML" || true
    if grep -q '^minimum-gas-prices' "$APP_TOML"; then
      sed -i 's/^minimum-gas-prices.*/minimum-gas-prices = "'"$MIN_GAS_PRICE"'"/' "$APP_TOML" || true
    else
      printf '\nminimum-gas-prices = "%s"\n' "$MIN_GAS_PRICE" >> "$APP_TOML"
    fi
  fi
  if [[ -f "$CONFIG_TOML" ]]; then
    sed -i 's|^\s*laddr = "tcp://127\.0\.0\.1:26657"|laddr = "tcp://0.0.0.0:26657"|' "$CONFIG_TOML" || true
    sed -i 's/^\s*addr_book_strict *=.*/addr_book_strict = false/' "$CONFIG_TOML" || true
    sed -i 's/^\s*cors_allowed_origins *=.*/cors_allowed_origins = ["*"]/' "$CONFIG_TOML" || true
  fi
}

# ── 실행 흐름 ────────────────────────────────────────────────────────────────
log "Starting setup…"
ensure_dirs
init_chain

# 1) 우선 file 백엔드로 시도
if ! "$APP" keys show "$KEY_NAME" --keyring-backend file --keyring-dir "$HOME_DIR" --home "$HOME_DIR" >/dev/null 2>&1; then
  create_key_file
fi

ADDRESS=""
if [[ -s /tmp/key.json ]]; then
  ADDRESS="$(jq -r '.address // empty' /tmp/key.json 2>/dev/null || true)"
fi

# file 키링 손상/읽기 실패 감지 시 클린업 후 재시도
if ! is_valid_addr "${ADDRESS:-}"; then
  # keys show로 에러 패턴 확인
  : > /tmp/keys_show.err
  keys_show_addr file >/dev/null 2>/tmp/keys_show.err || true
  if grep -qi -e "unmarshal" -e "UnmarshalBinaryLengthPrefixed" -e "Bytes left over" /tmp/keys_show.err 2>/dev/null; then
    recover_corrupted_file_ring
    create_key_file
    ADDRESS="$(jq -r '.address // empty' /tmp/key.json 2>/dev/null || true)"
  fi
fi

# 2) 여전히 실패면 test 백엔드로 폴백 (재시작 루프 차단)
EFFECTIVE_BACKEND="file"
if ! is_valid_addr "${ADDRESS:-}"; then
  log "file backend failed; falling back to test backend for bootstrap."
  create_key_test
  ADDRESS="$(jq -r '.address // empty' /tmp/key.json 2>/dev/null || true)"
  EFFECTIVE_BACKEND="test"
fi

# 주소 최종 확보 시도(마지막 방어막)
if ! is_valid_addr "${ADDRESS:-}"; then
  if ! ADDRESS="$(get_address_or_fail "$EFFECTIVE_BACKEND")"; then
    log "Address resolve failed; aborting."
    exit 1
  fi
fi
log "Address: $ADDRESS (backend=$EFFECTIVE_BACKEND)"

# 3) 제네시스/젠텍스
if ! grep -Fq "$ADDRESS" "$HOME_DIR/config/genesis.json"; then
  log "Adding genesis account: $GENESIS_COINS"
  "$APP" add-genesis-account "$ADDRESS" "$GENESIS_COINS" --home "$HOME_DIR"
fi

if ! ls -1 "$HOME_DIR/config/gentx"/gentx-*.json 1>/dev/null 2>&1; then
  log "Creating gentx (self-delegate: $SELF_DELEGATE)"
  gentx_with_backend "$EFFECTIVE_BACKEND"
else
  log "gentx already exists. Skipping."
fi

log "Collecting gentxs…"
"$APP" collect-gentxs --home "$HOME_DIR"

# 4) REST/gRPC/RPC 설정
configure_endpoints

# 5) validate & start
log "Validating genesis…"
"$APP" validate-genesis --home "$HOME_DIR"

log "Starting node…"
exec "$APP" start --home "$HOME_DIR" --minimum-gas-prices="$MIN_GAS_PRICE"
