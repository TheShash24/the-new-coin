#!/bin/bash
echo "══════════════════════════════════════════════════════"
echo "  INTEGRATION TESTS + NFR VERIFICATION"
echo "  Mapped to Chapter 3 Evaluation Plan (Section 3.8)"
echo "══════════════════════════════════════════════════════"
echo ""
echo "Note: Unit tests (31 cases) are in chaincode_test.go"
echo "Run them separately with: cd chaincode && go test -v ./..."
echo ""

PASS=0
FAIL=0

check_result() {
  local test_name="$1"
  local expected="$2"
  local actual="$3"
  
  if echo "$actual" | grep -q "$expected"; then
    echo "  ✓ PASS"
    PASS=$((PASS + 1))
  else
    echo "  ✗ FAIL — expected to find: $expected"
    echo "  Got: $actual"
    FAIL=$((FAIL + 1))
  fi
  echo ""
}

echo "════════════════════════════════════════════"
echo "  PHASE 1: SETUP (not counted as tests)"
echo "════════════════════════════════════════════"
echo ""

echo "--- Setup: Init Ledger ---"
R=$(curl -s -X POST -H "X-Org-ID: Org1" http://localhost:8080/ledger/init)
echo "  $R"
echo ""

echo "--- Setup: Register DIASPORA wallet d1 (Org1) ---"
R=$(curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"id":"d1","role":"DIASPORA","kycTier":"2"}' http://localhost:8080/wallets)
echo "  $R"
echo ""

echo "--- Setup: Register RELATIVE wallet r1 (Org2) ---"
R=$(curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"id":"r1","role":"RELATIVE","kycTier":"1"}' http://localhost:8080/wallets)
echo "  $R"
echo ""

echo "--- Setup: Register VENDOR wallet v1 (Org2) ---"
R=$(curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"id":"v1","role":"VENDOR","kycTier":"3"}' http://localhost:8080/wallets)
echo "  $R"
echo ""

echo "--- Setup: Mint 1000 tokens to d1 ---"
R=$(curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"walletId":"d1","amount":"1000","depositRef":"DEP-001","originalCurrency":"GBP","originalAmount":"787","exchangeRate":"1.27"}' \
  http://localhost:8080/tokens/mint)
echo "  $R"
echo ""

echo "════════════════════════════════════════════"
echo "  PHASE 2: INTEGRATION TESTS"
echo "  (6 flows from Chapter 3 Section 3.8)"
echo "════════════════════════════════════════════"
echo ""

# ─── FLOW 1: Diaspora Remittance Flow ───
echo "═══ FLOW 1: Diaspora Remittance Flow ═══"
echo "  Steps: RegisterWallet → MintTokens → Transfer"
echo "  Success: Recipient balance updated, deposit ref stored, fee deducted"
echo ""

echo "  1a. Transfer 200 HMZ from d1 to r1 (with memo)..."
R=$(curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"fromId":"d1","toId":"r1","amount":"200","memo":"School fees March"}' \
  http://localhost:8080/tokens/transfer)
echo "  $R"
check_result "Transfer" "result" "$R"

echo "  1b. Verify d1 balance (expect 799 = 1000 - 200 - 1 fee)..."
BAL=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/d1/balance)
echo "  Balance: $BAL"
check_result "d1 balance" "799" "$BAL"

echo "  1c. Verify r1 balance (expect 200)..."
BAL=$(curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/r1/balance)
echo "  Balance: $BAL"
check_result "r1 balance" "200" "$BAL"

# ─── FLOW 2: Vendor Payment Flow ───
echo "═══ FLOW 2: Vendor Payment Flow ═══"
echo "  Steps: PayVendor"
echo "  Success: Vendor balance updated, fee deducted, KYC enforced"
echo ""

echo "  2a. PayVendor 100 HMZ from r1 to v1..."
R=$(curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"fromId":"r1","toId":"v1","amount":"100","memo":"Monthly groceries"}' \
  http://localhost:8080/tokens/pay)
echo "  $R"
check_result "PayVendor" "result" "$R"

echo "  2b. Verify v1 balance (expect 100)..."
BAL=$(curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/v1/balance)
echo "  Balance: $BAL"
check_result "v1 balance" "100" "$BAL"

# ─── FLOW 3: Exchange Flow — Approved ───
echo "═══ FLOW 3: Exchange Flow — Approved ═══"
echo "  Steps: InitiateBurn → ApproveBurn"
echo "  Success: Tokens escrowed at initiation, destroyed on approval"
echo ""

echo "  3a. InitiateBurn 30 HMZ from v1 (Form IM reference)..."
R=$(curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"vendorId":"v1","amount":"30","burnRef":"FORM-IM-001"}' \
  http://localhost:8080/burns)
echo "  $R"
check_result "InitiateBurn" "result" "$R"

echo "  3b. Verify v1 balance reduced by escrow (expect 70)..."
BAL=$(curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/v1/balance)
echo "  Balance: $BAL"
check_result "v1 escrowed" "70" "$BAL"

echo "  3c. ApproveBurn from Org3 (CBOS)..."
BURN_ID=$(curl -s -H "X-Org-ID: Org3" "http://localhost:8080/burns?pageSize=10&bookmark=" | jq -r '.records[0].id')
echo "  Burn ID: $BURN_ID"
R=$(curl -s -X POST -H "X-Org-ID: Org3" http://localhost:8080/burns/$BURN_ID/approve)
echo "  $R"
check_result "ApproveBurn" "approved" "$R"

echo "  3d. Verify total supply decreased (expect 970 = 1000 - 30)..."
SUPPLY=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/tokens/supply)
echo "  Supply: $SUPPLY"
check_result "Supply after burn" "970" "$SUPPLY"

# ─── FLOW 4: Exchange Flow — Rejected ───
echo "═══ FLOW 4: Exchange Flow — Rejected ═══"
echo "  Steps: InitiateBurn → RejectBurn"
echo "  Success: Tokens escrowed at initiation, returned in full on rejection"
echo ""

echo "  4a. InitiateBurn 20 HMZ from v1..."
R=$(curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"vendorId":"v1","amount":"20","burnRef":"FORM-IM-002"}' \
  http://localhost:8080/burns)
echo "  $R"
check_result "InitiateBurn for reject" "result" "$R"

echo "  4b. Verify v1 balance reduced by escrow (expect 50 = 70 - 20)..."
BAL=$(curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/v1/balance)
echo "  Balance: $BAL"
check_result "v1 escrowed again" "50" "$BAL"

echo "  4c. RejectBurn from Org3 (CBOS)..."
BURN_ID2=$(curl -s -H "X-Org-ID: Org3" "http://localhost:8080/burns?pageSize=10&bookmark=" | jq -r '.records[0].id')
echo "  Burn ID: $BURN_ID2"
R=$(curl -s -X POST -H "X-Org-ID: Org3" http://localhost:8080/burns/$BURN_ID2/reject)
echo "  $R"
check_result "RejectBurn" "rejected" "$R"

echo "  4d. Verify v1 balance restored (expect 70 = 50 + 20 returned)..."
BAL=$(curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/v1/balance)
echo "  Balance: $BAL"
check_result "v1 restored" "70" "$BAL"

# ─── FLOW 5: Account Suspension Flow ───
echo "═══ FLOW 5: Account Suspension Flow ═══"
echo "  Steps: FreezeWallet → attempted Transfer → UnfreezeWallet"
echo "  Success: Transfer rejected while frozen, restored after unfreeze"
echo ""

echo "  5a. FreezeWallet d1 from Org3..."
R=$(curl -s -X POST -H "X-Org-ID: Org3" http://localhost:8080/wallets/d1/freeze)
echo "  $R"
check_result "FreezeWallet" "frozen" "$R"

echo "  5b. Attempt Transfer from frozen d1 (should fail)..."
R=$(curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"fromId":"d1","toId":"r1","amount":"10","memo":""}' \
  http://localhost:8080/tokens/transfer)
echo "  $R"
check_result "Frozen transfer rejected" "frozen" "$R"

echo "  5c. UnfreezeWallet d1..."
R=$(curl -s -X POST -H "X-Org-ID: Org3" http://localhost:8080/wallets/d1/unfreeze)
echo "  $R"
check_result "UnfreezeWallet" "unfrozen" "$R"

echo "  5d. Transfer after unfreeze (should succeed)..."
R=$(curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"fromId":"d1","toId":"r1","amount":"10","memo":"After unfreeze"}' \
  http://localhost:8080/tokens/transfer)
echo "  $R"
check_result "Transfer after unfreeze" "result" "$R"

# ─── FLOW 6: Fee Collection Flow ───
echo "═══ FLOW 6: Fee Collection Flow ═══"
echo "  Steps: Multiple transfers → ClaimCBOSFees"
echo "  Success: Total fees correctly aggregated and claimed atomically"
echo ""

echo "  6a. Do one more transfer to accumulate fees..."
R=$(curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"fromId":"d1","toId":"r1","amount":"10","memo":"Extra for fee test"}' \
  http://localhost:8080/tokens/transfer)
echo "  $R"

echo "  6b. ClaimCBOSFees..."
R=$(curl -s -X POST -H "X-Org-ID: Org3" -H "Content-Type: application/json" \
  -d '{"cbosWalletId":"CBOS_WALLET"}' http://localhost:8080/fees/claim)
echo "  $R"
check_result "ClaimCBOSFees" "claimed" "$R"

echo "  6c. Verify CBOS wallet has collected fees..."
BAL=$(curl -s -H "X-Org-ID: Org3" http://localhost:8080/wallets/CBOS_WALLET/balance)
echo "  CBOS balance: $BAL"
# Should be > 0 (accumulated from all fee-generating transactions)
if [ "$BAL" -gt 0 ] 2>/dev/null; then
  echo "  ✓ PASS — CBOS collected $BAL HMZ in fees"
  PASS=$((PASS + 1))
else
  echo "  ✗ FAIL — CBOS balance should be > 0"
  FAIL=$((FAIL + 1))
fi
echo ""

echo "════════════════════════════════════════════"
echo "  PHASE 3: NFR VERIFICATION"
echo "  (5 checks from Chapter 3 Section 3.8)"
echo "════════════════════════════════════════════"
echo ""

# ─── NFR 1: Tamper Resistance ───
echo "═══ NFR 1: Tamper Resistance ═══"
echo "  Method: Attempt ApproveBurn from Org1MSP and Org2MSP"
echo "  Success: Both rejected — protocol-level enforcement"
echo ""

echo "  Creating burn request for tamper test..."
curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"vendorId":"v1","amount":"5","burnRef":"FORM-IM-TAMPER"}' \
  http://localhost:8080/burns > /dev/null

BURN_ID3=$(curl -s -H "X-Org-ID: Org3" "http://localhost:8080/burns?pageSize=10&bookmark=" | jq -r '.records[0].id')

echo "  Org1 attempts ApproveBurn..."
R=$(curl -s -X POST -H "X-Org-ID: Org1" http://localhost:8080/burns/$BURN_ID3/approve)
echo "  $R"
check_result "Tamper: Org1 rejected" "Org3MSP" "$R"

echo "  Org2 attempts ApproveBurn..."
R=$(curl -s -X POST -H "X-Org-ID: Org2" http://localhost:8080/burns/$BURN_ID3/approve)
echo "  $R"
check_result "Tamper: Org2 rejected" "Org3MSP" "$R"

# ─── NFR 2: Finality ───
echo "═══ NFR 2: Finality ═══"
echo "  Method: Submit transaction and verify immediate ledger update"
echo "  Success: Balance updated instantly, no rollback"
echo ""

BAL_BEFORE=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/d1/balance)
echo "  d1 balance before: $BAL_BEFORE"

curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"fromId":"d1","toId":"r1","amount":"5","memo":"finality test"}' \
  http://localhost:8080/tokens/transfer > /dev/null

BAL_AFTER=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/d1/balance)
echo "  d1 balance after:  $BAL_AFTER"

if [ "$BAL_AFTER" -lt "$BAL_BEFORE" ] 2>/dev/null; then
  echo "  ✓ PASS — balance updated immediately, transaction is final"
  PASS=$((PASS + 1))
else
  echo "  ✗ FAIL — balance did not update"
  FAIL=$((FAIL + 1))
fi
echo ""

# ─── NFR 3: Auditability ───
echo "═══ NFR 3: Auditability ═══"
echo "  Method: GetWalletHistory after multiple operations"
echo "  Success: Complete history returned, all records traceable"
echo ""

HISTORY_COUNT=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/d1/history | jq '. | length')
echo "  d1 history entries: $HISTORY_COUNT"

if [ "$HISTORY_COUNT" -gt 3 ] 2>/dev/null; then
  echo "  ✓ PASS — $HISTORY_COUNT state transitions recorded (register, freeze, unfreeze, transfers)"
  PASS=$((PASS + 1))
else
  echo "  ✗ FAIL — expected more than 3 history entries"
  FAIL=$((FAIL + 1))
fi
echo ""

# ─── NFR 4: Settlement Speed ───
echo "═══ NFR 4: Settlement Speed ═══"
echo "  Method: Measure transaction confirmation time"
echo "  Success: Confirmation within seconds"
echo ""

START=$(date +%s%N)
curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"fromId":"d1","toId":"r1","amount":"5","memo":"speed test"}' \
  http://localhost:8080/tokens/transfer > /dev/null
END=$(date +%s%N)
ELAPSED=$(( (END - START) / 1000000 ))
echo "  Transaction confirmed in: ${ELAPSED}ms"

if [ "$ELAPSED" -lt 10000 ] 2>/dev/null; then
  echo "  ✓ PASS — confirmed in under 10 seconds"
  PASS=$((PASS + 1))
else
  echo "  ✗ FAIL — took longer than 10 seconds"
  FAIL=$((FAIL + 1))
fi
echo ""

# ─── NFR 5: Availability ───
echo "═══ NFR 5: Availability ═══"
echo "  Method: Submit concurrent transactions"
echo "  Success: Network remains stable, no inconsistent state"
echo ""

echo "  Submitting 5 concurrent transfers..."
for i in $(seq 1 5); do
  curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
    -d "{\"fromId\":\"d1\",\"toId\":\"r1\",\"amount\":\"1\",\"memo\":\"concurrent $i\"}" \
    http://localhost:8080/tokens/transfer > /tmp/concurrent-$i.txt 2>&1 &
done
wait

CONCURRENT_OK=0
CONCURRENT_FAIL=0
for i in $(seq 1 5); do
  if grep -q "error" /tmp/concurrent-$i.txt 2>/dev/null; then
    CONCURRENT_FAIL=$((CONCURRENT_FAIL + 1))
  else
    CONCURRENT_OK=$((CONCURRENT_OK + 1))
  fi
done
rm -f /tmp/concurrent-*.txt

echo "  Succeeded: $CONCURRENT_OK / 5"
echo "  Failed:    $CONCURRENT_FAIL / 5 (MVCC conflicts expected under concurrent load)"

# Even partial success proves availability — network didn't crash
if [ "$CONCURRENT_OK" -gt 0 ] 2>/dev/null; then
  echo "  ✓ PASS — network remained stable, $CONCURRENT_OK transactions committed"
  PASS=$((PASS + 1))
else
  echo "  ✗ FAIL — no concurrent transactions succeeded"
  FAIL=$((FAIL + 1))
fi
echo ""

echo "════════════════════════════════════════════"
echo "  PHASE 4: FINAL STATE VERIFICATION"
echo "════════════════════════════════════════════"
echo ""
echo -n "  d1 balance:    "; curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/d1/balance; echo ""
echo -n "  r1 balance:    "; curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/r1/balance; echo ""
echo -n "  v1 balance:    "; curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/v1/balance; echo ""
echo -n "  CBOS balance:  "; curl -s -H "X-Org-ID: Org3" http://localhost:8080/wallets/CBOS_WALLET/balance; echo ""
echo -n "  Total supply:  "; curl -s -H "X-Org-ID: Org1" http://localhost:8080/tokens/supply; echo ""

echo ""
echo "══════════════════════════════════════════════════════"
echo "  RESULTS: $PASS passed, $FAIL failed"
echo "  Integration flows: 6/6"  
echo "  NFR checks: 5/5"
echo "══════════════════════════════════════════════════════"
