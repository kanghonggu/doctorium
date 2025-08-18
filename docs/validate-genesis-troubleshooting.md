# Troubleshooting `doctoriumd validate-genesis`

If `doctoriumd validate-genesis` panics with a nil pointer, a faulty GenTx is usually the cause. The Docker entrypoint now pre-validates each GenTx and reports the failing index, but when running commands manually you can use these steps to isolate and fix the bad transaction:

## 1. Inspect GenTx entries
```bash
jq '.app_state.genutil.gen_txs[]' ~/.doctorium/config/genesis.json
```
Check each transaction for empty fields or missing validator information.

## 2. Decode each GenTx individually
```bash
mkdir ~/tmp-gentx && \
 jq -c '.app_state.genutil.gen_txs[]' ~/.doctorium/config/genesis.json \
   | nl -ba \
   | while read -r line; do
       num=$(echo "$line" | cut -f1)
       tx=$(echo "$line"  | cut -f2-)
       echo "$tx" > ~/tmp-gentx/$num.json
       doctoriumd tx decode ~/tmp-gentx/$num.json >/dev/null || echo "bad tx $num"
     done
```
Remove or regenerate any transaction reported as bad.

## 3. Rebuild and validate the genesis
```bash
doctoriumd init my-node
doctoriumd add-genesis-account youraddr 1000000000utoken
# Add or collect remaining GenTxs
doctoriumd validate-genesis
```
After replacing the faulty GenTxs, the validation should succeed without a panic.
