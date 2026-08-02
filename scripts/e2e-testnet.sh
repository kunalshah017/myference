#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
required=(MONAD_TESTNET_RPC_URL MYFERENCE_CONTRACT_ADDRESS MYFERENCE_DATABASE_URL MYFERENCE_BROKER_URL MYFERENCE_API_KEY CUSTOMER_ADDRESS PROVIDER_ADDRESS PROVIDER_PRIVATE_KEY PLATFORM_ADDRESS PLATFORM_PRIVATE_KEY WINDOWS_MACHINE_ID WINDOWS_STATUS_JSON OLLAMA_MODEL SESSION_ID CUSTOMER_DEPOSIT_TX PROVIDER_BOND_TX PROVIDER_SIGNER_TX OFFER_PUBLISH_TX SESSION_OPEN_TX)
for name in "${required[@]}"; do
  if [[ -z "${!name:-}" ]]; then echo "$name is required" >&2; exit 2; fi
done
for tool in cast curl jq psql awk shasum; do command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 2; }; done

case "$MONAD_TESTNET_RPC_URL" in *localhost*|*127.0.0.1*|*0.0.0.0*) echo "Anvil/localhost is forbidden for testnet acceptance" >&2; exit 2;; esac
[[ "$MYFERENCE_BROKER_URL" == https://* ]] || { echo "broker URL must use HTTPS" >&2; exit 2; }
for address in "$MYFERENCE_CONTRACT_ADDRESS" "$CUSTOMER_ADDRESS" "$PROVIDER_ADDRESS" "$PLATFORM_ADDRESS"; do
  [[ "$address" =~ ^0x[0-9a-fA-F]{40}$ && "$address" != "0x0000000000000000000000000000000000000000" ]] || { echo "invalid or zero EVM address: $address" >&2; exit 2; }
done
[[ "$(printf '%s' "$PROVIDER_ADDRESS" | tr '[:upper:]' '[:lower:]')" != "$(printf '%s' "$PLATFORM_ADDRESS" | tr '[:upper:]' '[:lower:]')" ]] || { echo "provider and platform claim addresses must be distinct" >&2; exit 2; }
[[ "$(jq -r '.goos' "$WINDOWS_STATUS_JSON")" == "windows" ]] || { echo "status evidence is not from the Windows CLI" >&2; exit 2; }
[[ "$(jq -r '.machine_id' "$WINDOWS_STATUS_JSON")" == "$WINDOWS_MACHINE_ID" ]] || { echo "Windows status machine mismatch" >&2; exit 2; }
[[ "$(jq -r '.backends' "$WINDOWS_STATUS_JSON")" -gt 0 ]] || { echo "Windows CLI reports no configured backends" >&2; exit 2; }
machine_signer="$(jq -r '.signer_address' "$WINDOWS_STATUS_JSON")"
attestation_message="$(jq -r '.attestation_message' "$WINDOWS_STATUS_JSON")"
attestation_signature="$(jq -r '.attestation_signature' "$WINDOWS_STATUS_JSON")"
cast wallet verify --address "$machine_signer" "$attestation_message" "$attestation_signature" >/dev/null || { echo "Windows status attestation signature is invalid" >&2; exit 2; }
[[ "$(jq -r '.version' "$WINDOWS_STATUS_JSON")" != "dev" && "$(jq -r '.commit' "$WINDOWS_STATUS_JSON")" == "$(git -C "$root" rev-parse HEAD)" ]] || { echo "Windows CLI is not the release build for this commit" >&2; exit 2; }
generated_at="$(jq -r '.generated_at' "$WINDOWS_STATUS_JSON")"
generated_epoch="$(date -u -d "$generated_at" +%s 2>/dev/null || date -u -j -f '%Y-%m-%dT%H:%M:%SZ' "$generated_at" +%s 2>/dev/null || true)"
[[ "$generated_epoch" =~ ^[0-9]+$ ]] || { echo "Windows status has an invalid generation time" >&2; exit 2; }
status_age=$(( $(date -u +%s) - generated_epoch ))
[[ "$status_age" -ge 0 && "$status_age" -le 300 ]] || { echo "Windows status must be generated within the last five minutes" >&2; exit 2; }
capacity_payload="$(jq -r '.capacity_payload | @base64d' "$WINDOWS_STATUS_JSON")"
capacity_hash="$(printf '%s' "$capacity_payload" | shasum -a 256 | awk '{print $1}')"
[[ "$capacity_hash" == "$(jq -r '.capacity_sha256' "$WINDOWS_STATUS_JSON")" ]] || { echo "signed Windows capacity digest mismatch" >&2; exit 2; }
[[ "$(jq -c '.capacity' "$WINDOWS_STATUS_JSON")" == "$capacity_payload" ]] || { echo "Windows capacity payload mismatch" >&2; exit 2; }
expected_attestation="myference-status:v2:$WINDOWS_MACHINE_ID:windows:$(jq -r '.goarch' "$WINDOWS_STATUS_JSON"):$(jq -r '.version' "$WINDOWS_STATUS_JSON"):$(jq -r '.commit' "$WINDOWS_STATUS_JSON"):$generated_at:$capacity_hash"
[[ "$attestation_message" == "$expected_attestation" ]] || { echo "Windows attestation fields are not bound to its signature" >&2; exit 2; }
[[ "$(jq -r '.capacity.available' "$WINDOWS_STATUS_JSON")" -gt 0 ]] || { echo "Windows CLI reports no live capacity" >&2; exit 2; }
offer_id="$(jq -r --arg model "$OLLAMA_MODEL" '.offers[] | select(.model==$model) | .offer_id' <<< "$capacity_payload" | head -1)"
[[ -n "$offer_id" && "$offer_id" != "null" ]] || { echo "Windows capacity contains no matching real model" >&2; exit 2; }
model_hash="$(jq -r --arg offer "$offer_id" '.offers[] | select(.offer_id==$offer) | .model_hash' <<< "$capacity_payload" | head -1)"
capability_hash="$(jq -r --arg offer "$offer_id" '.offers[] | select(.offer_id==$offer) | .capability_hash' <<< "$capacity_payload" | head -1)"
price_version="$(jq -r --arg offer "$offer_id" '.offers[] | select(.offer_id==$offer) | .price_version' <<< "$capacity_payload" | head -1)"
backend_kind="$(jq -r --arg offer "$offer_id" '.offers[] | select(.offer_id==$offer) | .backend_kind' <<< "$capacity_payload" | head -1)"
[[ "$model_hash" =~ ^0x[0-9a-fA-F]{64}$ && "$capability_hash" =~ ^0x[0-9a-fA-F]{64}$ && "$price_version" =~ ^[1-9][0-9]*$ ]] || { echo "Windows offer proof keys are malformed" >&2; exit 2; }
[[ "$backend_kind" == "ollama" ]] || { echo "testnet acceptance requires a live Windows Ollama backend" >&2; exit 2; }
[[ "$(cast chain-id --rpc-url "$MONAD_TESTNET_RPC_URL")" == "10143" ]] || { echo "RPC is not Monad testnet chain 10143" >&2; exit 2; }
[[ "$(cast wallet address --private-key "$PROVIDER_PRIVATE_KEY" | tr '[:upper:]' '[:lower:]')" == "$(printf '%s' "$PROVIDER_ADDRESS" | tr '[:upper:]' '[:lower:]')" ]] || { echo "provider key/address mismatch" >&2; exit 2; }
[[ "$(cast wallet address --private-key "$PLATFORM_PRIVATE_KEY" | tr '[:upper:]' '[:lower:]')" == "$(printf '%s' "$PLATFORM_ADDRESS" | tr '[:upper:]' '[:lower:]')" ]] || { echo "platform key/address mismatch" >&2; exit 2; }
[[ "$(cast code "$MYFERENCE_CONTRACT_ADDRESS" --rpc-url "$MONAD_TESTNET_RPC_URL")" != "0x" ]] || { echo "contract has no testnet bytecode" >&2; exit 2; }
[[ "$(curl --fail --silent "$MYFERENCE_BROKER_URL/healthz")" == "ok" ]] || { echo "hosted broker is unhealthy" >&2; exit 2; }

verify_tx() {
  local hash="$1" label="$2" expected_sender="${3:-}" expected_to="${4:-}" receipt status actual_sender actual_to
  receipt="$(cast receipt "$hash" --rpc-url "$MONAD_TESTNET_RPC_URL" --json)"
  status="$(jq -r '.status' <<< "$receipt")"
  [[ "$status" == "0x1" || "$status" == "1" ]] || { echo "$label transaction failed: $hash" >&2; exit 1; }
  if [[ -n "$expected_sender" ]]; then
    actual_sender="$(jq -r '.from' <<< "$receipt" | tr '[:upper:]' '[:lower:]')"
    [[ "$actual_sender" == "$(printf '%s' "$expected_sender" | tr '[:upper:]' '[:lower:]')" ]] || { echo "$label sender mismatch" >&2; exit 1; }
  fi
  if [[ -n "$expected_to" ]]; then
    actual_to="$(jq -r '.to' <<< "$receipt" | tr '[:upper:]' '[:lower:]')"
    [[ "$actual_to" == "$(printf '%s' "$expected_to" | tr '[:upper:]' '[:lower:]')" ]] || { echo "$label destination mismatch" >&2; exit 1; }
  fi
}
verify_tx "$CUSTOMER_DEPOSIT_TX" customer-deposit "$CUSTOMER_ADDRESS" "$MYFERENCE_CONTRACT_ADDRESS"
verify_tx "$PROVIDER_BOND_TX" provider-bond "$PROVIDER_ADDRESS" "$MYFERENCE_CONTRACT_ADDRESS"
verify_tx "$PROVIDER_SIGNER_TX" provider-signer "$PROVIDER_ADDRESS" "$MYFERENCE_CONTRACT_ADDRESS"
verify_tx "$OFFER_PUBLISH_TX" offer-publish "$PROVIDER_ADDRESS" "$MYFERENCE_CONTRACT_ADDRESS"
verify_tx "$SESSION_OPEN_TX" session-open "$CUSTOMER_ADDRESS" "$MYFERENCE_CONTRACT_ADDRESS"
setup_transactions=("$CUSTOMER_DEPOSIT_TX" "$PROVIDER_BOND_TX" "$PROVIDER_SIGNER_TX" "$OFFER_PUBLISH_TX" "$SESSION_OPEN_TX")
[[ "$(printf '%s\n' "${setup_transactions[@]}" | sort -u | wc -l | tr -d ' ')" == "${#setup_transactions[@]}" ]] || { echo "setup transaction hashes must be distinct" >&2; exit 2; }
[[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'customerBalances(address)(uint256)' "$CUSTOMER_ADDRESS" --rpc-url "$MONAD_TESTNET_RPC_URL")" != "0" ]] || { echo "customer deposit is not reflected on-chain" >&2; exit 1; }
[[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'providerBonds(address)(uint256)' "$PROVIDER_ADDRESS" --rpc-url "$MONAD_TESTNET_RPC_URL")" != "0" ]] || { echo "provider bond is not reflected on-chain" >&2; exit 1; }
[[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'providerSigners(address,address)(bool)' "$PROVIDER_ADDRESS" "$machine_signer" --rpc-url "$MONAD_TESTNET_RPC_URL")" == "true" ]] || { echo "Windows machine signer is not authorized on-chain" >&2; exit 1; }
offer_hash="$(cast keccak "$offer_id")"
verify_call() {
  local hash="$1" signature="$2" input selector
  input="$(cast tx "$hash" --rpc-url "$MONAD_TESTNET_RPC_URL" --json | jq -r '.input')"
  selector="$(cast sig "$signature")"
  [[ "${input:0:10}" == "$selector" ]] || { echo "transaction $hash is not $signature" >&2; exit 1; }
  printf '%s' "$input"
}
deposit_input="$(verify_call "$CUSTOMER_DEPOSIT_TX" 'deposit()')"
bond_input="$(verify_call "$PROVIDER_BOND_TX" 'depositBond()')"
[[ "$deposit_input" == "$(cast sig 'deposit()')" && "$bond_input" == "$(cast sig 'depositBond()')" ]] || { echo "deposit setup calldata is malformed" >&2; exit 1; }
deposit_value="$(cast to-dec "$(cast tx "$CUSTOMER_DEPOSIT_TX" --rpc-url "$MONAD_TESTNET_RPC_URL" --json | jq -r '.value')")"
bond_value="$(cast to-dec "$(cast tx "$PROVIDER_BOND_TX" --rpc-url "$MONAD_TESTNET_RPC_URL" --json | jq -r '.value')")"
[[ "$deposit_value" != "0" && "$bond_value" != "0" ]] || { echo "deposit setup transactions must transfer native MON" >&2; exit 1; }
signer_input="$(verify_call "$PROVIDER_SIGNER_TX" 'setProviderSigner(address,bool)')"
signer_decoded="$(cast decode-calldata 'setProviderSigner(address,bool)' "$signer_input")"
printf '%s' "$signer_decoded" | grep -qi "$machine_signer" && printf '%s' "$signer_decoded" | grep -qi 'true' || { echo "provider signer transaction does not authorize this machine" >&2; exit 1; }
offer_input="$(verify_call "$OFFER_PUBLISH_TX" 'publishOffer(bytes32,bytes32,bytes32,uint256,uint256,uint256)')"
offer_decoded="$(cast decode-calldata 'publishOffer(bytes32,bytes32,bytes32,uint256,uint256,uint256)' "$offer_input")"
for expected in "$offer_hash" "$model_hash" "$capability_hash"; do printf '%s' "$offer_decoded" | grep -qi "$expected" || { echo "offer transaction argument mismatch" >&2; exit 1; }; done
session_input="$(verify_call "$SESSION_OPEN_TX" 'openSession(bytes32,uint256,uint64)')"
cast decode-calldata 'openSession(bytes32,uint256,uint64)' "$session_input" | grep -qi "$SESSION_ID" || { echo "session transaction ID mismatch" >&2; exit 1; }
[[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'latestOfferVersion(address,bytes32)(uint64)' "$PROVIDER_ADDRESS" "$offer_hash" --rpc-url "$MONAD_TESTNET_RPC_URL")" != "0" ]] || { echo "Windows offer is not published on-chain" >&2; exit 1; }
published_offer="$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'offerVersions(address,bytes32,uint64)(bool,bytes32,bytes32,uint64,uint256,uint256,uint256)' "$PROVIDER_ADDRESS" "$offer_hash" "$price_version" --rpc-url "$MONAD_TESTNET_RPC_URL")"
printf '%s' "$published_offer" | grep -qi "$model_hash" || { echo "published model hash does not match Windows capacity" >&2; exit 1; }
printf '%s' "$published_offer" | grep -qi "$capability_hash" || { echo "published capability hash does not match Windows capacity" >&2; exit 1; }
cast call "$MYFERENCE_CONTRACT_ADDRESS" 'sessions(bytes32)(address,uint256,uint256,uint64,uint64,bool)' "$SESSION_ID" --rpc-url "$MONAD_TESTNET_RPC_URL" | grep -qi "$CUSTOMER_ADDRESS" || { echo "customer session is not open on-chain" >&2; exit 1; }

route_count="$(psql "$MYFERENCE_DATABASE_URL" -v machine="$WINDOWS_MACHINE_ID" -v model="$OLLAMA_MODEL" -v offer="$offer_id" -v provider="$PROVIDER_ADDRESS" -Atc "SELECT count(*) FROM provider_routing_state prs JOIN machines m ON m.id=prs.machine_id JOIN accounts a ON a.id=m.account_id WHERE prs.machine_id=:'machine' AND prs.model=:'model' AND prs.offer_id=:'offer' AND prs.backend_kind='ollama' AND lower(a.wallet_address)=lower(:'provider') AND prs.confirmed_bond AND prs.healthy AND prs.capacity>0")"
[[ "$route_count" -gt 0 ]] || { echo "the real Windows Ollama route is not currently eligible" >&2; exit 1; }

work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT
jq -n --arg model "$OLLAMA_MODEL" '{model:$model,stream:true,messages:[{role:"user",content:"Return one short sentence proving live inference."}]}' > "$work/request.json"
curl --fail --silent --show-error --no-buffer -D "$work/headers" -o "$work/body" \
  -H "Authorization: Bearer $MYFERENCE_API_KEY" -H 'Content-Type: application/json' -H 'X-Myference-Max-Spend: 1000000000000000000' \
  --data-binary @"$work/request.json" "$MYFERENCE_BROKER_URL/v1/chat/completions"
grep -q '^data: ' "$work/body" || { echo "broker returned no streamed inference" >&2; exit 1; }
grep -q 'data: \[DONE\]' "$work/body" || { echo "stream did not complete" >&2; exit 1; }
openai_request_id="$(awk 'BEGIN{IGNORECASE=1}/^X-Request-ID:/{gsub("\r","");print $2}' "$work/headers" | tail -1)"
[[ "$openai_request_id" =~ ^0x[0-9a-fA-F]{64}$ ]] || { echo "broker returned an invalid OpenAI request ID" >&2; exit 1; }

jq -n --arg model "$OLLAMA_MODEL" '{model:$model,max_tokens:64,stream:true,messages:[{role:"user",content:"Return one different short sentence proving live inference."}]}' > "$work/anthropic-request.json"
curl --fail --silent --show-error --no-buffer -D "$work/anthropic-headers" -o "$work/anthropic-body" \
  -H "x-api-key: $MYFERENCE_API_KEY" -H 'anthropic-version: 2023-06-01' -H 'Content-Type: application/json' -H 'X-Myference-Max-Spend: 1000000000000000000' \
  --data-binary @"$work/anthropic-request.json" "$MYFERENCE_BROKER_URL/v1/messages"
grep -q '^event: message_start' "$work/anthropic-body" || { echo "Anthropic stream did not start" >&2; exit 1; }
grep -q '^event: content_block_delta' "$work/anthropic-body" || { echo "Anthropic stream returned no inference" >&2; exit 1; }
grep -q '^event: message_stop' "$work/anthropic-body" || { echo "Anthropic stream did not complete" >&2; exit 1; }
anthropic_request_id="$(awk 'BEGIN{IGNORECASE=1}/^X-Request-ID:/{gsub("\r","");print $2}' "$work/anthropic-headers" | tail -1)"
[[ "$anthropic_request_id" =~ ^0x[0-9a-fA-F]{64}$ && "$anthropic_request_id" != "$openai_request_id" ]] || { echo "broker returned an invalid Anthropic request ID" >&2; exit 1; }

wait_settlement() {
  local request_id="$1" state="" settlement_tx="" routed_session="" row amounts provider_amount fee_amount total_charge input_tokens output_tokens compute_milliseconds
  for _ in $(seq 1 90); do
    row="$(psql "$MYFERENCE_DATABASE_URL" -v request="$request_id" -AtF '|' -c "SELECT r.state,COALESCE(sq.transaction_hash,''),r.session_id FROM requests r LEFT JOIN settlement_queue sq ON sq.request_id=r.id WHERE r.id=:'request'")"
    IFS='|' read -r state settlement_tx routed_session <<< "$row"
    [[ "$state" == "settled" && -n "$settlement_tx" ]] && break
    sleep 2
  done
  [[ "$state" == "settled" ]] || { echo "request $request_id did not reach indexed settlement; last state=$state" >&2; exit 1; }
  [[ "$(printf '%s' "$routed_session" | tr '[:upper:]' '[:lower:]')" == "$(printf '%s' "$SESSION_ID" | tr '[:upper:]' '[:lower:]')" ]] || { echo "request used an unexpected spending session" >&2; exit 1; }
  verify_tx "$settlement_tx" inference-settlement
  [[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'settledRequests(bytes32)(bool)' "$request_id" --rpc-url "$MONAD_TESTNET_RPC_URL")" == "true" ]] || { echo "request $request_id is not settled in the contract" >&2; exit 1; }
  amounts="$(psql "$MYFERENCE_DATABASE_URL" -v request="$request_id" -AtF '|' -c "SELECT cs.provider_amount::text,cs.fee_amount::text,(cs.provider_amount+cs.fee_amount)::text,rp.input_tokens::text,rp.output_tokens::text,rp.compute_milliseconds::text FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id WHERE cs.request_id=:'request'")"
  IFS='|' read -r provider_amount fee_amount total_charge input_tokens output_tokens compute_milliseconds <<< "$amounts"
  [[ "$provider_amount" =~ ^[0-9]+$ && "$fee_amount" =~ ^[0-9]+$ && "$total_charge" =~ ^[0-9]+$ && "$input_tokens" =~ ^[0-9]+$ && "$output_tokens" =~ ^[0-9]+$ && "$compute_milliseconds" =~ ^[0-9]+$ ]] || { echo "indexed settlement usage or amounts missing" >&2; exit 1; }
  printf '%s|%s|%s|%s|%s|%s|%s\n' "$settlement_tx" "$provider_amount" "$fee_amount" "$total_charge" "$input_tokens" "$output_tokens" "$compute_milliseconds"
}

IFS='|' read -r openai_settlement_tx openai_provider_amount openai_fee_amount openai_total_charge openai_input_tokens openai_output_tokens openai_compute_ms <<< "$(wait_settlement "$openai_request_id")"
IFS='|' read -r anthropic_settlement_tx anthropic_provider_amount anthropic_fee_amount anthropic_total_charge anthropic_input_tokens anthropic_output_tokens anthropic_compute_ms <<< "$(wait_settlement "$anthropic_request_id")"
provider_claim_tx="$(cast send "$MYFERENCE_CONTRACT_ADDRESS" 'claim()' --private-key "$PROVIDER_PRIVATE_KEY" --rpc-url "$MONAD_TESTNET_RPC_URL" --json | jq -r '.transactionHash')"
platform_claim_tx="$(cast send "$MYFERENCE_CONTRACT_ADDRESS" 'claim()' --private-key "$PLATFORM_PRIVATE_KEY" --rpc-url "$MONAD_TESTNET_RPC_URL" --json | jq -r '.transactionHash')"
verify_tx "$provider_claim_tx" provider-claim "$PROVIDER_ADDRESS"
verify_tx "$platform_claim_tx" platform-claim "$PLATFORM_ADDRESS"
[[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'claimable(address)(uint256)' "$PROVIDER_ADDRESS" --rpc-url "$MONAD_TESTNET_RPC_URL")" == "0" ]] || { echo "provider claimable balance was not withdrawn" >&2; exit 1; }
[[ "$(cast call "$MYFERENCE_CONTRACT_ADDRESS" 'claimable(address)(uint256)' "$PLATFORM_ADDRESS" --rpc-url "$MONAD_TESTNET_RPC_URL")" == "0" ]] || { echo "platform claimable balance was not withdrawn" >&2; exit 1; }
explorer="${MYFERENCE_EXPLORER_URL:-https://testnet.monadexplorer.com}"
commit="$(git -C "$root" rev-parse HEAD)"
cli_version="$(jq -r '.version + " (" + .commit + ")"' "$WINDOWS_STATUS_JSON")"

cat > "$root/docs/demo.md" <<EOF
# Myference Monad Testnet Evidence

- Verified at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
- Repository commit: $commit
- Contract: [$MYFERENCE_CONTRACT_ADDRESS]($explorer/address/$MYFERENCE_CONTRACT_ADDRESS)
- Windows CLI: $cli_version; machine $WINDOWS_MACHINE_ID
- Real provider model: $OLLAMA_MODEL
- OpenAI request ID: $openai_request_id
- Anthropic request ID: $anthropic_request_id
- Customer deposit: [$CUSTOMER_DEPOSIT_TX]($explorer/tx/$CUSTOMER_DEPOSIT_TX)
- Provider bond: [$PROVIDER_BOND_TX]($explorer/tx/$PROVIDER_BOND_TX)
- Machine signer authorization: [$PROVIDER_SIGNER_TX]($explorer/tx/$PROVIDER_SIGNER_TX)
- Immutable offer: [$OFFER_PUBLISH_TX]($explorer/tx/$OFFER_PUBLISH_TX)
- Spending session: [$SESSION_OPEN_TX]($explorer/tx/$SESSION_OPEN_TX)
- OpenAI settlement: [$openai_settlement_tx]($explorer/tx/$openai_settlement_tx)
- OpenAI charge/provider/fee: $openai_total_charge / $openai_provider_amount / $openai_fee_amount wei MON
- OpenAI measured usage: $openai_input_tokens input tokens / $openai_output_tokens output tokens / $openai_compute_ms ms
- Anthropic settlement: [$anthropic_settlement_tx]($explorer/tx/$anthropic_settlement_tx)
- Anthropic charge/provider/fee: $anthropic_total_charge / $anthropic_provider_amount / $anthropic_fee_amount wei MON
- Anthropic measured usage: $anthropic_input_tokens input tokens / $anthropic_output_tokens output tokens / $anthropic_compute_ms ms
- Provider claim: [$provider_claim_tx]($explorer/tx/$provider_claim_tx)
- Platform claim: [$platform_claim_tx]($explorer/tx/$platform_claim_tx)

The acceptance script generated this file from RPC receipts, broker streaming, and indexed PostgreSQL state. It intentionally excludes API keys, private keys, prompts, and model output.
EOF

echo "real Monad testnet inference settled: $explorer/tx/$openai_settlement_tx and $explorer/tx/$anthropic_settlement_tx"
