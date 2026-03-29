# Project Context: Blockchain-Based Diaspora Remittances Platform for Sudan

## What This Is

A master's thesis project building a permissioned blockchain platform for diaspora remittances to Sudan, implemented on Hyperledger Fabric 2.5 with Go chaincode. The thesis follows the Design Science Research (DSR) paradigm combined with Requirements Engineering as the development methodology.

---

## The Problem

Three Sudan-specific dimensions that no existing system solves simultaneously:

1. **Sanctions inaccessibility:** Traditional correspondent banking and systems built on its infrastructure (PayPal, Wise) are largely inaccessible for Sudan due to the country's sanctions history. Even where accessible, the per-transaction settlement model introduces significant cost and delay.

2. **Value leakage:** Cross-border transfers are dominated by Hawala and informal intermediaries. Hard currency enters Sudan but immediately leaves the formal economy — CBOS cannot retain it or use it to manage FX reserves.

3. **No protocol-level regulatory enforcement:** Existing systems enforce compliance at the application layer, which can be bypassed. CBOS has no real-time visibility or structural authority over flows.

**Research contribution statement:** *"The artefact demonstrates that all three dimensions can be addressed simultaneously by creating a blockchain-based system that enforces a closed-loop environment that retains hard currency and preserves its value, eliminates the need for correspondent banking and similar infrastructure thus reducing cost and time, and maps Fabric's organisational identity model to Sudan's regulatory hierarchy, enforcing compliance at the protocol level rather than the application layer."*

---

## Network Architecture

**Three-organisation model:**

- **Org1MSP — Bank A (diaspora-side, UK):** peer0.org1 (port 7051), ca-org1. Handles diaspora sender registration, deposit confirmation, and token minting.
- **Org2MSP — Bank B Sudan (Sudan-side):** peer0.org2 (port 9051), ca-org2. Handles recipient and vendor registration, in-country operations.
- **Org3MSP — CBOS (Sudan's central bank):** Raft orderer (port 7050) + ca-org3 ONLY. No endorsing peer. CBOS operates the ordering service — every transaction passes through CBOS infrastructure before commitment.

**Channel:** mychannel
**State database:** CouchDB per peer
**Endorsement policy:** AND('Org1MSP.peer', 'Org2MSP.peer')
**Chaincode:** Single chaincode on mychannel, written in Go using contractapi framework, 22 functions

**Why CBOS is orderer-only:** This maps precisely to CBOS's real-world supervisory role — full visibility over all flows without co-approving routine commercial activity. CBOS's regulatory authority over specific functions is enforced via Org3MSP client-identity checks inside the chaincode, separate from the endorsement policy.

---

## Token Model: Deposit Tokens

**Unit of value:** Deposit tokens — not cryptocurrency, not stablecoins, not CBDC.

- Tokens are minted only against a real, verified bank deposit. Every token has a traceable, permanent deposit reference.
- The backing is held at a regulated bank (Bank A), not managed by an algorithm.
- 1:1 backing ratio maintained architecturally — MintTokens requires a depositRef, ApproveBurn is the only mechanism that reduces supply.
- **There is no withdrawal function.** The only exit is through a CBOS-approved burn flow backed by a Ministry of Trade import document (Form IM).
- Hard currency retention is architectural, not token-level — the closed-loop model means hard currency deposited in the UK stays within the formal banking system until CBOS approves a release.

**Settlement model:** Periodic net settlement — the blockchain shared ledger enables trusted net position calculation, reducing USD movement to a single periodic transfer between banks rather than per-transaction correspondent banking.

---

## Key Design Decisions

| Decision | Selected | Rejected (and why) |
|----------|----------|-------------------|
| Unit of value | Deposit tokens | Cryptocurrency (volatile, no traceability), Stablecoins (reserve-pool trust model weaker than deposit-level traceability), CBDC (Sudan has no operational CBDC) |
| Blockchain platform | Hyperledger Fabric | Public blockchains (permissionless, anonymous), Quorum (Ethereum address-based identity, insufficient for CBOS mapping), R3 Corda (point-to-point model, CBOS system-wide visibility architecturally impossible) |
| Exit model | Closed-loop | Open model (value leakage), Partially open model (threshold exploitable, fragmentation attack possible) |
| Settlement | Periodic net settlement | Per-transaction correspondent banking, RTGS (both: per-transaction cost burden) |
| Endorsement policy | AND(Org1MSP.peer, Org2MSP.peer) | Single-bank endorsement (unilateral trust gap) |
| CBOS position | Ordering service (Org3MSP) | Endorsing peer (conflates supervisory with operational), Read-only access (visibility without enforcement) |
| Smart contract language | Go 1.21 | Java (JVM overhead), Node.js (dynamic typing, float precision risk) |
| State database | CouchDB | LevelDB (key-value only, cannot support rich queries for AML monitoring) |

---

## Data Structures (On-Chain)

### Wallet — Key: WALLET:{walletID}
| Field | Type | Purpose |
|-------|------|---------|
| balance | int64 | Current token balance. Cannot go below zero |
| docType | string | Always 'wallet' |
| frozen | bool | Set exclusively by FreezeWallet, gated on Org3MSP |
| id | string | Unique wallet identifier |
| kycTier | int | 1, 2, or 3. Enforced on every PayVendor call |
| owner | string | X.509 Distinguished Name. Compared against GetClientIdentity().GetID() |
| role | string | DIASPORA, RELATIVE, VENDOR, or CBOS |
| updatedAt | string | GetTxTimestamp() — deterministic across all peers |

### Transaction — Key: TX:{txID}
| Field | Type | Purpose |
|-------|------|---------|
| amount | int64 | Token amount |
| depositRef | string | Populated on MintTokens only. Permanent deposit reference |
| docType | string | Always 'transaction' |
| fee | int64 | CBOS fee deducted |
| from | string | Source wallet ID |
| participants | []string | Array of from and to wallet IDs. Enables $elemMatch CouchDB query |
| timestamp | string | GetTxTimestamp() — deterministic from signed proposal |
| to | string | Destination wallet ID |
| txId | string | GetTxID() — 64-character hex |
| txType | string | MINT, TRANSFER, PAYMENT, ESCROW, BURN, or REFUND |

### BurnRequest — Key: BURN:{txID}
| Field | Type | Purpose |
|-------|------|---------|
| amount | int64 | Tokens escrowed at InitiateBurn. Refunded exactly if rejected |
| burnRef | string | Ministry of Trade Form IM invoice reference. Mandatory |
| docType | string | Always 'burnRequest' |
| id | string | TX ID of the InitiateBurn transaction |
| initiatedAt | string | GetTxTimestamp() of InitiateBurn |
| resolvedAt | string | GetTxTimestamp() of resolution. Empty while PENDING |
| status | string | PENDING, APPROVED, or REJECTED |
| vendorId | string | Wallet ID of the requesting vendor |

### FeeEntry — Key: Composite key CBOS_FEE:{txID}
| Field | Type | Purpose |
|-------|------|---------|
| amount | int64 | Fee in tokens |
| docType | string | Always 'fee' |
| source | string | Wallet ID from which fee was collected |
| timestamp | string | GetTxTimestamp() of parent transaction |
| txId | string | GetTxID() — composite key suffix guaranteeing uniqueness |

---

## Chaincode Functions (22 total)

### Diaspora Flow
| Function | Caller | Purpose |
|----------|--------|---------|
| RegisterWallet | Org1MSP or Org2MSP | Creates wallet with role, KYC tier, X.509 identity |
| MintTokens | Org1MSP only | Mints tokens against mandatory deposit reference |
| Transfer | DIASPORA wallet owner | Transfers to RELATIVE wallet, deducts fee, enforces KYC tier |

### Sudan-Side Flow
| Function | Caller | Purpose |
|----------|--------|---------|
| PayVendor | RELATIVE wallet owner | Pays vendor, deducts fee, enforces KYC tier |
| InitiateBurn | VENDOR wallet owner | Creates burn request with Form IM reference, escrows tokens immediately |

### CBOS Regulatory Flow
| Function | Caller | Purpose |
|----------|--------|---------|
| ApproveBurn | Org3MSP only | Destroys escrowed tokens (not transferred), resolves BurnRequest |
| RejectBurn | Org3MSP only | Returns escrowed tokens in full |
| FreezeWallet | Org3MSP only | Blocks all operations on wallet |
| UnfreezeWallet | Org3MSP only | Reverses freeze |
| ClaimCBOSFees | Org3MSP only | Aggregates all FeeEntry records, credits CBOS, deletes records atomically |

### Audit and Query
| Function | Caller | Purpose |
|----------|--------|---------|
| GetWalletHistory | Any | GetHistoryForKey — reads immutable chain, not CouchDB |
| GetTransactionsByWallet | Any | CouchDB Mango query, paginated, sorted by timestamp |
| GetLargeTransactions | Org3MSP | CouchDB — AML large-transaction monitoring |
| GetPendingBurnRequests | Org3MSP | CouchDB — CBOS exchange request review queue |

---

## Four Protocol-Level Bugs Identified and Fixed

**Bug 1 — Non-deterministic timestamps:** time.Now() inside chaincode causes ENDORSEMENT_MISMATCH across peers. Fix: ctx.GetStub().GetTxTimestamp().

**Bug 2 — MVCC hot-key on CBOS wallet:** All concurrent transactions read WALLET:CBOS at version N. Transaction 1 commits. All others rejected. Fix: FeeEntry struct with composite key CBOS_FEE:{txID} — no two transactions share a key. CBOS sweeps all keys via GetStateByPartialCompositeKey at claim time.

**Bug 3 — Double-spend in burn flow:** Tokens not locked during CBOS review window. Fix: Escrow at InitiateBurn — tokens deducted from vendor balance immediately at submission, before approval.

**Bug 4 — CouchDB $or sort failure:** $or queries cannot be combined with sort in CouchDB Mango. Fix: participants []string array + $elemMatch query on composite index.

---

## CouchDB Indexes
| Index File | Fields | Supports |
|------------|--------|---------|
| indexTxParticipants.json | docType, participants, timestamp | GetTransactionsByWallet |
| indexTxTimestamp.json | docType, timestamp | General timestamp-sorted lookups |
| indexTxAmount.json | docType, amount, timestamp | GetLargeTransactions — AML monitoring |
| indexBurnStatus.json | docType, status, initiatedAt | GetPendingBurnRequests |
| indexWalletRole.json | docType, role | GetWalletsByRole |

---

## Technology Stack
| Component | Technology |
|-----------|------------|
| Blockchain Platform | Hyperledger Fabric 2.5 |
| Smart Contract Language | Go 1.21 |
| Contract Framework | contractapi |
| State Database | CouchDB |
| Certificate Management | Hyperledger Fabric CA |
| Development Environment | WSL2 Ubuntu 24, Docker, VS Code |
| Test Network | Fabric test-network (3-org) |

---

## Legal Framework (Sudan)

Four instruments that the system must comply with:

1. **CBOS Regulations on Electronic Payment Services (c. 2020):** Deposit traceability, CBOS supervisory authority over electronic payment systems.
2. **Foreign Exchange Dealing Act 1981 (as amended):** All foreign exchange release must go through authorised channels. The closed-loop model is an architectural interpretation of this — requires Sudanese legal counsel confirmation before production deployment.
3. **AML/CFT Law of 2014 + CBOS Circular No. 8/2014:** Supervisory authority, unusual transaction monitoring, account freezing powers.
4. **Electronic Transactions Act 2007:** Electronic identities uniquely attributable, tamper-evident records legally recognised.

---

## What Has Been Completed

### Phase 1: Code Review and Fixes (DONE)
All 13 findings identified and applied:

**Critical (6):**
- C1: requireOrg2() renamed to requireCBOS(), MSP check changed to Org3MSP
- C2: depositRef uniqueness enforced in MintTokens via composite key
- C3: Silent error in GetWalletHistory fixed
- C4: Transfer now checks DIASPORA role on sender, RELATIVE role on recipient
- C5: PayVendor now checks RELATIVE role on payer
- C6: RegisterWallet accepts Org1 OR Org2

**Medium (4):**
- M1: Transfer checks recipient frozen status
- M2: Added ESCROW and REFUND txTypes with "SYSTEM_ESCROW" address
- M3: Integer overflow check on Transfer
- M4: Comment cleanup

**Low (3):**
- L1: Empty string validation added across 11 functions
- L2: Constants for docType strings
- L3: Guard preventing CBOS wallet from being frozen

---

## What Needs to Be Built Next

1. **Phase 2: REST API** — Using Hyperledger Fabric Gateway SDK (Go), expose all 22 chaincode functions as HTTP endpoints with three org identity contexts
2. **Phase 3: Simple web frontend** — Role-based testing interface for all four flows
3. **Unit tests** — 15 specific test cases using Fabric chaincode mock library
4. **Integration testing** — Six end-to-end flows
5. **Deployment to 3-org test network**

---

## Files Available
- chaincode.go — 850+ lines, 22 functions, all bugs and review findings fixed
- go.mod — Go module file
- 5 CouchDB index JSON files in META-INF/statedb/couchdb/indexes/
