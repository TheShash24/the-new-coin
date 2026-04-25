#!/bin/bash
echo "══════════════════════════════════════════════════════"
echo "  PERFORMANCE TESTS"
echo "  Preliminary latency assessment (single-machine)"
echo "══════════════════════════════════════════════════════"
echo ""

# Setup
echo "Setting up test wallets..."
curl -s -X POST -H "X-Org-ID: Org1" http://localhost:8080/ledger/init > /dev/null 2>&1
curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"id":"perf-d1","role":"DIASPORA","kycTier":"3"}' http://localhost:8080/wallets > /dev/null 2>&1
curl -s -X POST -H "X-Org-ID: Org2" -H "Content-Type: application/json" \
  -d '{"id":"perf-r1","role":"RELATIVE","kycTier":"3"}' http://localhost:8080/wallets > /dev/null 2>&1
curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
  -d '{"walletId":"perf-d1","amount":"100000","depositRef":"PERF-001","originalCurrency":"USD","originalAmount":"100000","exchangeRate":"1.00"}' \
  http://localhost:8080/tokens/mint > /dev/null 2>&1
echo "Setup complete."
echo ""

# Test 1: Write latency (Transfer)
echo "── Write Latency: 10 sequential transfers ──"
TOTAL=0
for i in $(seq 1 10); do
  START=$(date +%s%N)
  curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
    -d "{\"fromId\":\"perf-d1\",\"toId\":\"perf-r1\",\"amount\":\"10\",\"memo\":\"perf $i\"}" \
    http://localhost:8080/tokens/transfer > /dev/null
  END=$(date +%s%N)
  ELAPSED=$(( (END - START) / 1000000 ))
  TOTAL=$((TOTAL + ELAPSED))
  echo "  Transfer $i: ${ELAPSED}ms"
done
WRITE_AVG=$((TOTAL / 10))
echo "  ─────────────────────────"
echo "  Average write latency: ${WRITE_AVG}ms"
echo ""

# Test 2: Read latency (GetBalance)
echo "── Read Latency: 10 balance queries ──"
TOTAL=0
for i in $(seq 1 10); do
  START=$(date +%s%N)
  curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/perf-d1/balance > /dev/null
  END=$(date +%s%N)
  ELAPSED=$(( (END - START) / 1000000 ))
  TOTAL=$((TOTAL + ELAPSED))
  echo "  Query $i: ${ELAPSED}ms"
done
READ_AVG=$((TOTAL / 10))
echo "  ─────────────────────────"
echo "  Average read latency: ${READ_AVG}ms"
echo ""

# Test 3: Concurrent writes
echo "── Concurrent Load: 5 simultaneous transfers ──"
echo "  (validates MVCC hot-key fix — Bug 2)"
START=$(date +%s%N)
for i in $(seq 1 5); do
  curl -s -X POST -H "X-Org-ID: Org1" -H "Content-Type: application/json" \
    -d "{\"fromId\":\"perf-d1\",\"toId\":\"perf-r1\",\"amount\":\"5\",\"memo\":\"concurrent $i\"}" \
    http://localhost:8080/tokens/transfer > /tmp/perf-$i.txt 2>&1 &
done
wait
END=$(date +%s%N)
CONC_ELAPSED=$(( (END - START) / 1000000 ))

OK=0; FAILED=0
for i in $(seq 1 5); do
  if grep -q "error" /tmp/perf-$i.txt 2>/dev/null; then
    FAILED=$((FAILED + 1))
  else
    OK=$((OK + 1))
  fi
done
rm -f /tmp/perf-*.txt
echo "  Completed in: ${CONC_ELAPSED}ms"
echo "  Succeeded: $OK / 5"
echo "  Failed: $FAILED / 5"
echo ""

# Final verification
echo "── Final Balance Verification ──"
D1=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/wallets/perf-d1/balance)
R1=$(curl -s -H "X-Org-ID: Org2" http://localhost:8080/wallets/perf-r1/balance)
SUP=$(curl -s -H "X-Org-ID: Org1" http://localhost:8080/tokens/supply)
echo "  perf-d1: $D1"
echo "  perf-r1: $R1"
echo "  Supply:  $SUP"
echo ""

echo "══════════════════════════════════════════════════════"
echo "  SUMMARY"
echo "  Write latency (avg):    ${WRITE_AVG}ms"
echo "  Read latency (avg):     ${READ_AVG}ms"  
echo "  Concurrent (5 writes):  ${CONC_ELAPSED}ms ($OK/5 succeeded)"
echo "══════════════════════════════════════════════════════"
