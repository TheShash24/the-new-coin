package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ============================================================
// Constants — L2 fix: all docType strings defined as constants
// ============================================================

const (
	DocTypeWallet      = "wallet"
	DocTypeTransaction = "transaction"
	DocTypeBurnRequest = "burnRequest"
	DocTypeFee         = "fee"
)

const (
	KeyPrefixWallet  = "WALLET:"
	KeyPrefixTx      = "TX:"
	KeyPrefixBurn    = "BURN:"
	KeyPrefixFee     = "CBOS_FEE"   // composite key object type (no colon — Fabric adds delimiters)
	KeyPrefixDeposit = "DEPOSIT_REF" // composite key object type for depositRef dedup
	KeyTotalSupply   = "TOTAL_SUPPLY"
)

const (
	RoleDiaspora = "DIASPORA"
	RoleRelative = "RELATIVE"
	RoleVendor   = "VENDOR"
	RoleCBOS     = "CBOS"
)

// txTypes — M2 fix: ESCROW and REFUND added
const (
	TxTypeMint     = "MINT"
	TxTypeTransfer = "TRANSFER"
	TxTypePayment  = "PAYMENT"
	TxTypeEscrow   = "ESCROW"
	TxTypeBurn     = "BURN"
	TxTypeRefund   = "REFUND"
)

const (
	BurnStatusPending  = "PENDING"
	BurnStatusApproved = "APPROVED"
	BurnStatusRejected = "REJECTED"
)

const (
	MSPOrg1 = "Org1MSP"
	MSPOrg2 = "Org2MSP"
	MSPOrg3 = "Org3MSP" // CBOS — orderer-only org
)

// M2 fix: SYSTEM_ESCROW is the virtual address used in ESCROW and REFUND transactions
const (
	SystemEscrow = "SYSTEM_ESCROW"
	CBOSWalletID = "CBOS_WALLET"
)

// Fee policy: 10 basis points (0.10%), minimum 1 token
const (
	FeeRateBasisPoints int64 = 10
	FeeRateDivisor     int64 = 10000
)

// KYC tier per-transaction limits (CBOS policy — configurable)
const (
	KYCTier1MaxAmount int64 = 1_000
	KYCTier2MaxAmount int64 = 10_000
	// Tier 3: no limit
)

// ============================================================
// Data Structures
// ============================================================

// Wallet — key: WALLET:{walletID}
type Wallet struct {
	Balance   int64  `json:"balance"`
	DocType   string `json:"docType"`
	Frozen    bool   `json:"frozen"`
	ID        string `json:"id"`
	KYCTier   int    `json:"kycTier"`
	Owner     string `json:"owner"`    // X.509 identity string from GetClientIdentity().GetID()
	Role      string `json:"role"`
	UpdatedAt string `json:"updatedAt"` // RFC3339 from GetTxTimestamp()
}

// Transaction — key: TX:{txID}
type Transaction struct {
	Amount       int64    `json:"amount"`
	DepositRef   string   `json:"depositRef"`  // populated on MINT only
	DocType      string   `json:"docType"`
	Fee          int64    `json:"fee"`
	From         string   `json:"from"`
	Participants []string `json:"participants"` // enables $elemMatch CouchDB query — Bug 4 fix
	Timestamp    string   `json:"timestamp"`
	To           string   `json:"to"`
	TxID         string   `json:"txId"`
	TxType       string   `json:"txType"`
}

// BurnRequest — key: BURN:{txID}
type BurnRequest struct {
	Amount      int64  `json:"amount"`
	BurnRef     string `json:"burnRef"`     // Ministry of Trade Form IM reference
	DocType     string `json:"docType"`
	ID          string `json:"id"`          // TxID of the InitiateBurn transaction
	InitiatedAt string `json:"initiatedAt"`
	ResolvedAt  string `json:"resolvedAt"`  // empty while PENDING
	Status      string `json:"status"`
	VendorID    string `json:"vendorId"`
}

// FeeEntry — key: composite CBOS_FEE:{txID}
// Bug 2 fix: fees are stored as individual FeeEntry records rather than directly
// crediting the CBOS wallet, eliminating the MVCC hot-key conflict on concurrent transactions.
type FeeEntry struct {
	Amount    int64  `json:"amount"`
	DocType   string `json:"docType"`
	Source    string `json:"source"`    // wallet ID from which fee was collected
	Timestamp string `json:"timestamp"`
	TxID      string `json:"txId"`
}

// ============================================================
// Response wrappers for query/history functions
// ============================================================

type WalletHistoryEntry struct {
	TxID      string  `json:"txId"`
	Timestamp string  `json:"timestamp"`
	IsDelete  bool    `json:"isDelete"`
	Record    *Wallet `json:"record"` // nil for deletion records
}

type PaginatedTransactionResult struct {
	Records             []*Transaction `json:"records"`
	FetchedRecordsCount int32          `json:"fetchedRecordsCount"`
	Bookmark            string         `json:"bookmark"`
}

type PaginatedBurnResult struct {
	Records             []*BurnRequest `json:"records"`
	FetchedRecordsCount int32          `json:"fetchedRecordsCount"`
	Bookmark            string         `json:"bookmark"`
}

type PaginatedWalletResult struct {
	Records             []*Wallet `json:"records"`
	FetchedRecordsCount int32     `json:"fetchedRecordsCount"`
	Bookmark            string    `json:"bookmark"`
}

// ============================================================
// SmartContract
// ============================================================

type SmartContract struct {
	contractapi.Contract
}

// ============================================================
// Package-level helpers
// ============================================================

func requireOrg1(ctx contractapi.TransactionContextInterface) error {
	mspid, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSP ID: %w", err)
	}
	if mspid != MSPOrg1 {
		return fmt.Errorf("caller must be Org1MSP (Bank A): got %s", mspid)
	}
	return nil
}

// requireCBOS checks that the caller belongs to Org3MSP (CBOS).
// C1 fix: the original code checked Org2MSP here — corrected to Org3MSP.
func requireCBOS(ctx contractapi.TransactionContextInterface) error {
	mspid, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSP ID: %w", err)
	}
	if mspid != MSPOrg3 {
		return fmt.Errorf("caller must be Org3MSP (CBOS): got %s", mspid)
	}
	return nil
}

// requireOrg1OrOrg2 allows Bank A (diaspora-side) or Bank B (Sudan-side) to call.
// C6 fix: RegisterWallet was previously restricted to one org.
func requireOrg1OrOrg2(ctx contractapi.TransactionContextInterface) error {
	mspid, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSP ID: %w", err)
	}
	if mspid != MSPOrg1 && mspid != MSPOrg2 {
		return fmt.Errorf("caller must be Org1MSP or Org2MSP: got %s", mspid)
	}
	return nil
}

func getCallerID(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity: %w", err)
	}
	return id, nil
}

// getTxTimestamp returns the transaction timestamp as an RFC3339 string.
// Bug 1 fix: time.Now() is non-deterministic across peers and causes ENDORSEMENT_MISMATCH.
// GetTxTimestamp() is derived from the signed proposal and is identical on all peers.
func getTxTimestamp(ctx contractapi.TransactionContextInterface) (string, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get transaction timestamp: %w", err)
	}
	return ts.AsTime().Format(time.RFC3339), nil
}

func walletKey(id string) string { return KeyPrefixWallet + id }
func txKey(id string) string     { return KeyPrefixTx + id }
func burnKey(id string) string   { return KeyPrefixBurn + id }

// calculateFee computes the CBOS supervisory fee at 10 basis points (0.10%), minimum 1 token.
func calculateFee(amount int64) int64 {
	fee := (amount * FeeRateBasisPoints) / FeeRateDivisor
	if fee < 1 {
		fee = 1
	}
	return fee
}

// loadWallet reads and unmarshals a wallet from ledger state.
func loadWallet(ctx contractapi.TransactionContextInterface, walletID string) (*Wallet, error) {
	data, err := ctx.GetStub().GetState(walletKey(walletID))
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet %s: %w", walletID, err)
	}
	if data == nil {
		return nil, fmt.Errorf("wallet %s does not exist", walletID)
	}
	var w Wallet
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet %s: %w", walletID, err)
	}
	return &w, nil
}

// saveWallet marshals and writes a wallet to ledger state.
func saveWallet(ctx contractapi.TransactionContextInterface, w *Wallet) error {
	data, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet %s: %w", w.ID, err)
	}
	return ctx.GetStub().PutState(walletKey(w.ID), data)
}

// saveTx marshals and writes a transaction record to ledger state.
func saveTx(ctx contractapi.TransactionContextInterface, tx *Transaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}
	return ctx.GetStub().PutState(txKey(tx.TxID), data)
}

// saveFeeEntry writes a composite-keyed FeeEntry record.
// Bug 2 fix: instead of crediting the CBOS wallet directly (which creates an MVCC hot-key
// conflict when concurrent transfers all read WALLET:CBOS at version N), each transaction
// writes to its own unique composite key. CBOS sweeps them all in ClaimCBOSFees.
func saveFeeEntry(ctx contractapi.TransactionContextInterface, txID string, amount int64, source string, ts string) error {
	feeKey, err := ctx.GetStub().CreateCompositeKey(KeyPrefixFee, []string{txID})
	if err != nil {
		return fmt.Errorf("failed to create fee composite key: %w", err)
	}
	entry := FeeEntry{
		Amount:    amount,
		DocType:   DocTypeFee,
		Source:    source,
		Timestamp: ts,
		TxID:      txID,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal fee entry: %w", err)
	}
	return ctx.GetStub().PutState(feeKey, data)
}

// updateTotalSupply adjusts the TOTAL_SUPPLY counter by delta (+mint / -burn).
// Note: MintTokens is restricted to Org1MSP; concurrent mints are operationally
// sequential (one deposit reference per transaction), so MVCC conflict risk is low.
func updateTotalSupply(ctx contractapi.TransactionContextInterface, delta int64) error {
	var current int64
	data, err := ctx.GetStub().GetState(KeyTotalSupply)
	if err != nil {
		return fmt.Errorf("failed to read total supply: %w", err)
	}
	if data != nil {
		if err := json.Unmarshal(data, &current); err != nil {
			return fmt.Errorf("failed to unmarshal total supply: %w", err)
		}
	}
	current += delta
	if current < 0 {
		return fmt.Errorf("total supply cannot go negative")
	}
	updated, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("failed to marshal total supply: %w", err)
	}
	return ctx.GetStub().PutState(KeyTotalSupply, updated)
}

// ============================================================
// 1. InitLedger
// ============================================================

// InitLedger creates the CBOS system wallet at chaincode instantiation.
// Prototype note: guarded by requireOrg1 because Org3MSP (CBOS) has no endorsing peer
// and cannot satisfy the AND(Org1MSP.peer, Org2MSP.peer) endorsement policy. In a
// production deployment this would be a jointly-bootstrapped transaction.
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	if err := requireOrg1(ctx); err != nil {
		return err
	}
	// Idempotency guard — safe to call again on chaincode upgrade
	existing, err := ctx.GetStub().GetState(walletKey(CBOSWalletID))
	if err != nil {
		return fmt.Errorf("failed to check CBOS wallet: %w", err)
	}
	if existing != nil {
		return nil
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	cbos := &Wallet{
		Balance:   0,
		DocType:   DocTypeWallet,
		Frozen:    false,
		ID:        CBOSWalletID,
		KYCTier:   3,
		Owner:     "CBOS_SYSTEM", // sentinel: CBOS has no endorsing peer, no real X.509 DN here
		Role:      RoleCBOS,
		UpdatedAt: ts,
	}
	return saveWallet(ctx, cbos)
}

// ============================================================
// 2. RegisterWallet
// ============================================================

// RegisterWallet creates a new wallet with the caller's X.509 identity as owner.
// C6 fix: accepts Org1MSP (diaspora-side) or Org2MSP (Sudan-side) callers.
func (s *SmartContract) RegisterWallet(ctx contractapi.TransactionContextInterface, id string, role string, kycTier int) error {
	// L1: empty string validation
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}
	if role == "" {
		return fmt.Errorf("role cannot be empty")
	}
	// C6 fix: was restricted to a single org
	if err := requireOrg1OrOrg2(ctx); err != nil {
		return err
	}
	if role != RoleDiaspora && role != RoleRelative && role != RoleVendor && role != RoleCBOS {
		return fmt.Errorf("invalid role %q: must be DIASPORA, RELATIVE, VENDOR, or CBOS", role)
	}
	if kycTier < 1 || kycTier > 3 {
		return fmt.Errorf("invalid kycTier %d: must be 1, 2, or 3", kycTier)
	}
	existing, err := ctx.GetStub().GetState(walletKey(id))
	if err != nil {
		return fmt.Errorf("failed to check wallet existence: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("wallet %s already exists", id)
	}
	owner, err := getCallerID(ctx)
	if err != nil {
		return err
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	w := &Wallet{
		Balance:   0,
		DocType:   DocTypeWallet,
		Frozen:    false,
		ID:        id,
		KYCTier:   kycTier,
		Owner:     owner,
		Role:      role,
		UpdatedAt: ts,
	}
	return saveWallet(ctx, w)
}

// ============================================================
// 3. MintTokens
// ============================================================

// MintTokens credits tokens to a wallet against a verified bank deposit.
// Restricted to Org1MSP (Bank A) — all deposits originate on the diaspora side.
// C2 fix: depositRef uniqueness enforced via composite key to prevent double-minting.
func (s *SmartContract) MintTokens(ctx contractapi.TransactionContextInterface, walletID string, amount int64, depositRef string) error {
	// L1: empty string validation
	if walletID == "" {
		return fmt.Errorf("walletID cannot be empty")
	}
	if depositRef == "" {
		return fmt.Errorf("depositRef cannot be empty")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if err := requireOrg1(ctx); err != nil {
		return err
	}
	// C2 fix: composite key ensures each depositRef can only be used once
	depKey, err := ctx.GetStub().CreateCompositeKey(KeyPrefixDeposit, []string{depositRef})
	if err != nil {
		return fmt.Errorf("failed to create deposit ref key: %w", err)
	}
	existing, err := ctx.GetStub().GetState(depKey)
	if err != nil {
		return fmt.Errorf("failed to check deposit ref: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("depositRef %q has already been used", depositRef)
	}
	if err := ctx.GetStub().PutState(depKey, []byte{0x00}); err != nil {
		return fmt.Errorf("failed to record deposit ref: %w", err)
	}
	w, err := loadWallet(ctx, walletID)
	if err != nil {
		return err
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	w.Balance += amount
	w.UpdatedAt = ts
	if err := saveWallet(ctx, w); err != nil {
		return err
	}
	if err := updateTotalSupply(ctx, amount); err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()
	tx := &Transaction{
		Amount:       amount,
		DepositRef:   depositRef,
		DocType:      DocTypeTransaction,
		Fee:          0,
		From:         "",
		Participants: []string{walletID},
		Timestamp:    ts,
		To:           walletID,
		TxID:         txID,
		TxType:       TxTypeMint,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 4. Transfer
// ============================================================

// Transfer moves tokens from a DIASPORA wallet to a RELATIVE wallet, deducting a CBOS fee.
// C4 fix: enforces role constraints — sender must be DIASPORA, recipient must be RELATIVE.
// M1 fix: recipient frozen status checked.
// M3 fix: integer overflow guard on amount + fee.
func (s *SmartContract) Transfer(ctx contractapi.TransactionContextInterface, fromID string, toID string, amount int64) error {
	// L1: empty string validation
	if fromID == "" {
		return fmt.Errorf("fromID cannot be empty")
	}
	if toID == "" {
		return fmt.Errorf("toID cannot be empty")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	fromWallet, err := loadWallet(ctx, fromID)
	if err != nil {
		return err
	}
	toWallet, err := loadWallet(ctx, toID)
	if err != nil {
		return err
	}
	// C4 fix: role checks
	if fromWallet.Role != RoleDiaspora {
		return fmt.Errorf("sender wallet must have DIASPORA role")
	}
	if toWallet.Role != RoleRelative {
		return fmt.Errorf("recipient wallet must have RELATIVE role")
	}
	// Ownership check
	callerID, err := getCallerID(ctx)
	if err != nil {
		return err
	}
	if fromWallet.Owner != callerID {
		return fmt.Errorf("caller does not own wallet %s", fromID)
	}
	if fromWallet.Frozen {
		return fmt.Errorf("sender wallet %s is frozen", fromID)
	}
	// M1 fix: check recipient frozen status
	if toWallet.Frozen {
		return fmt.Errorf("recipient wallet %s is frozen", toID)
	}
	fee := calculateFee(amount)
	// M3 fix: integer overflow check
	if amount > math.MaxInt64-fee {
		return fmt.Errorf("amount + fee overflows int64")
	}
	total := amount + fee
	if fromWallet.Balance < total {
		return fmt.Errorf("insufficient balance: have %d, need %d", fromWallet.Balance, total)
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	fromWallet.Balance -= total
	toWallet.Balance += amount
	fromWallet.UpdatedAt = ts
	toWallet.UpdatedAt = ts
	if err := saveWallet(ctx, fromWallet); err != nil {
		return err
	}
	if err := saveWallet(ctx, toWallet); err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()
	// Bug 2 fix: fee stored as FeeEntry composite key, not direct credit to CBOS wallet
	if err := saveFeeEntry(ctx, txID, fee, fromID, ts); err != nil {
		return err
	}
	tx := &Transaction{
		Amount:       amount,
		DocType:      DocTypeTransaction,
		Fee:          fee,
		From:         fromID,
		Participants: []string{fromID, toID},
		Timestamp:    ts,
		To:           toID,
		TxID:         txID,
		TxType:       TxTypeTransfer,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 5. PayVendor
// ============================================================

// PayVendor transfers tokens from a RELATIVE wallet to a VENDOR wallet.
// C5 fix: enforces RELATIVE role on payer.
// KYC tier limits: Tier 1 ≤ 1,000 tokens, Tier 2 ≤ 10,000 tokens, Tier 3 unlimited.
func (s *SmartContract) PayVendor(ctx contractapi.TransactionContextInterface, fromID string, toID string, amount int64) error {
	// L1: empty string validation
	if fromID == "" {
		return fmt.Errorf("fromID cannot be empty")
	}
	if toID == "" {
		return fmt.Errorf("toID cannot be empty")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	fromWallet, err := loadWallet(ctx, fromID)
	if err != nil {
		return err
	}
	toWallet, err := loadWallet(ctx, toID)
	if err != nil {
		return err
	}
	// C5 fix: role check
	if fromWallet.Role != RoleRelative {
		return fmt.Errorf("payer wallet must have RELATIVE role")
	}
	if toWallet.Role != RoleVendor {
		return fmt.Errorf("recipient wallet must have VENDOR role")
	}
	// Ownership check
	callerID, err := getCallerID(ctx)
	if err != nil {
		return err
	}
	if fromWallet.Owner != callerID {
		return fmt.Errorf("caller does not own wallet %s", fromID)
	}
	// KYC tier enforcement
	switch fromWallet.KYCTier {
	case 1:
		if amount > KYCTier1MaxAmount {
			return fmt.Errorf("KYC tier 1 limit exceeded: max %d tokens per transaction", KYCTier1MaxAmount)
		}
	case 2:
		if amount > KYCTier2MaxAmount {
			return fmt.Errorf("KYC tier 2 limit exceeded: max %d tokens per transaction", KYCTier2MaxAmount)
		}
	case 3:
		// no limit
	default:
		return fmt.Errorf("invalid KYC tier %d on wallet %s", fromWallet.KYCTier, fromID)
	}
	if fromWallet.Frozen {
		return fmt.Errorf("payer wallet %s is frozen", fromID)
	}
	if toWallet.Frozen {
		return fmt.Errorf("vendor wallet %s is frozen", toID)
	}
	fee := calculateFee(amount)
	// M3 fix: integer overflow check
	if amount > math.MaxInt64-fee {
		return fmt.Errorf("amount + fee overflows int64")
	}
	total := amount + fee
	if fromWallet.Balance < total {
		return fmt.Errorf("insufficient balance: have %d, need %d", fromWallet.Balance, total)
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	fromWallet.Balance -= total
	toWallet.Balance += amount
	fromWallet.UpdatedAt = ts
	toWallet.UpdatedAt = ts
	if err := saveWallet(ctx, fromWallet); err != nil {
		return err
	}
	if err := saveWallet(ctx, toWallet); err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()
	if err := saveFeeEntry(ctx, txID, fee, fromID, ts); err != nil {
		return err
	}
	tx := &Transaction{
		Amount:       amount,
		DocType:      DocTypeTransaction,
		Fee:          fee,
		From:         fromID,
		Participants: []string{fromID, toID},
		Timestamp:    ts,
		To:           toID,
		TxID:         txID,
		TxType:       TxTypePayment,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 6. InitiateBurn
// ============================================================

// InitiateBurn submits a foreign-exchange burn request backed by a Form IM reference.
// Bug 3 fix: tokens are escrowed immediately at submission (balance deducted before
// CBOS review) to prevent double-spend during the approval window.
// M2 fix: txType ESCROW, to field set to SYSTEM_ESCROW.
func (s *SmartContract) InitiateBurn(ctx contractapi.TransactionContextInterface, vendorID string, amount int64, burnRef string) error {
	// L1: empty string validation
	if vendorID == "" {
		return fmt.Errorf("vendorID cannot be empty")
	}
	if burnRef == "" {
		return fmt.Errorf("burnRef cannot be empty")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	vendorWallet, err := loadWallet(ctx, vendorID)
	if err != nil {
		return err
	}
	if vendorWallet.Role != RoleVendor {
		return fmt.Errorf("wallet %s must have VENDOR role", vendorID)
	}
	callerID, err := getCallerID(ctx)
	if err != nil {
		return err
	}
	if vendorWallet.Owner != callerID {
		return fmt.Errorf("caller does not own wallet %s", vendorID)
	}
	if vendorWallet.Frozen {
		return fmt.Errorf("wallet %s is frozen", vendorID)
	}
	if vendorWallet.Balance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", vendorWallet.Balance, amount)
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	// Bug 3 fix: deduct tokens immediately — escrow before CBOS review
	vendorWallet.Balance -= amount
	vendorWallet.UpdatedAt = ts
	if err := saveWallet(ctx, vendorWallet); err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()
	burn := &BurnRequest{
		Amount:      amount,
		BurnRef:     burnRef,
		DocType:     DocTypeBurnRequest,
		ID:          txID,
		InitiatedAt: ts,
		ResolvedAt:  "",
		Status:      BurnStatusPending,
		VendorID:    vendorID,
	}
	burnData, err := json.Marshal(burn)
	if err != nil {
		return fmt.Errorf("failed to marshal burn request: %w", err)
	}
	if err := ctx.GetStub().PutState(burnKey(txID), burnData); err != nil {
		return fmt.Errorf("failed to save burn request: %w", err)
	}
	// M2 fix: txType ESCROW; to = SYSTEM_ESCROW; participants includes both wallet IDs
	tx := &Transaction{
		Amount:       amount,
		DocType:      DocTypeTransaction,
		Fee:          0,
		From:         vendorID,
		Participants: []string{vendorID, SystemEscrow},
		Timestamp:    ts,
		To:           SystemEscrow,
		TxID:         txID,
		TxType:       TxTypeEscrow,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 7. ApproveBurn
// ============================================================

// ApproveBurn destroys the escrowed tokens and marks the burn request APPROVED.
// Tokens were already deducted from the vendor wallet at InitiateBurn — this call
// permanently removes them from circulation. No wallet is credited.
func (s *SmartContract) ApproveBurn(ctx contractapi.TransactionContextInterface, burnTxID string) error {
	// L1: empty string validation
	if burnTxID == "" {
		return fmt.Errorf("burnTxID cannot be empty")
	}
	if err := requireCBOS(ctx); err != nil {
		return err
	}
	burnData, err := ctx.GetStub().GetState(burnKey(burnTxID))
	if err != nil {
		return fmt.Errorf("failed to read burn request: %w", err)
	}
	if burnData == nil {
		return fmt.Errorf("burn request %s does not exist", burnTxID)
	}
	var burn BurnRequest
	if err := json.Unmarshal(burnData, &burn); err != nil {
		return fmt.Errorf("failed to unmarshal burn request: %w", err)
	}
	if burn.Status != BurnStatusPending {
		return fmt.Errorf("burn request %s is not PENDING (status: %s)", burnTxID, burn.Status)
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	burn.Status = BurnStatusApproved
	burn.ResolvedAt = ts
	updatedBurnData, err := json.Marshal(burn)
	if err != nil {
		return fmt.Errorf("failed to marshal burn request: %w", err)
	}
	if err := ctx.GetStub().PutState(burnKey(burnTxID), updatedBurnData); err != nil {
		return fmt.Errorf("failed to update burn request: %w", err)
	}
	// Tokens are permanently destroyed — decrement total supply
	if err := updateTotalSupply(ctx, -burn.Amount); err != nil {
		return err
	}
	newTxID := ctx.GetStub().GetTxID()
	tx := &Transaction{
		Amount:       burn.Amount,
		DocType:      DocTypeTransaction,
		Fee:          0,
		From:         SystemEscrow,
		Participants: []string{burn.VendorID, SystemEscrow},
		Timestamp:    ts,
		To:           "",
		TxID:         newTxID,
		TxType:       TxTypeBurn,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 8. RejectBurn
// ============================================================

// RejectBurn returns the escrowed tokens to the vendor and marks the request REJECTED.
// M2 fix: txType REFUND, from = SYSTEM_ESCROW.
func (s *SmartContract) RejectBurn(ctx contractapi.TransactionContextInterface, burnTxID string) error {
	// L1: empty string validation
	if burnTxID == "" {
		return fmt.Errorf("burnTxID cannot be empty")
	}
	if err := requireCBOS(ctx); err != nil {
		return err
	}
	burnData, err := ctx.GetStub().GetState(burnKey(burnTxID))
	if err != nil {
		return fmt.Errorf("failed to read burn request: %w", err)
	}
	if burnData == nil {
		return fmt.Errorf("burn request %s does not exist", burnTxID)
	}
	var burn BurnRequest
	if err := json.Unmarshal(burnData, &burn); err != nil {
		return fmt.Errorf("failed to unmarshal burn request: %w", err)
	}
	if burn.Status != BurnStatusPending {
		return fmt.Errorf("burn request %s is not PENDING (status: %s)", burnTxID, burn.Status)
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	// Return escrowed tokens in full to vendor
	vendorWallet, err := loadWallet(ctx, burn.VendorID)
	if err != nil {
		return err
	}
	vendorWallet.Balance += burn.Amount
	vendorWallet.UpdatedAt = ts
	if err := saveWallet(ctx, vendorWallet); err != nil {
		return err
	}
	burn.Status = BurnStatusRejected
	burn.ResolvedAt = ts
	updatedBurnData, err := json.Marshal(burn)
	if err != nil {
		return fmt.Errorf("failed to marshal burn request: %w", err)
	}
	if err := ctx.GetStub().PutState(burnKey(burnTxID), updatedBurnData); err != nil {
		return fmt.Errorf("failed to update burn request: %w", err)
	}
	newTxID := ctx.GetStub().GetTxID()
	// M2 fix: REFUND txType, from = SYSTEM_ESCROW, to = vendor
	tx := &Transaction{
		Amount:       burn.Amount,
		DocType:      DocTypeTransaction,
		Fee:          0,
		From:         SystemEscrow,
		Participants: []string{burn.VendorID, SystemEscrow},
		Timestamp:    ts,
		To:           burn.VendorID,
		TxID:         newTxID,
		TxType:       TxTypeRefund,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 9. FreezeWallet
// ============================================================

// FreezeWallet suspends all operations on a wallet. Org3MSP (CBOS) authority required.
// AML/CFT Law of 2014 — CBOS supervisory power to freeze accounts.
// L3 fix: guard prevents the CBOS system wallet from being frozen.
func (s *SmartContract) FreezeWallet(ctx contractapi.TransactionContextInterface, walletID string) error {
	// L1: empty string validation
	if walletID == "" {
		return fmt.Errorf("walletID cannot be empty")
	}
	if err := requireCBOS(ctx); err != nil {
		return err
	}
	w, err := loadWallet(ctx, walletID)
	if err != nil {
		return err
	}
	// L3 fix: CBOS wallet must never be frozen
	if w.Role == RoleCBOS {
		return fmt.Errorf("CBOS wallet cannot be frozen")
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	w.Frozen = true
	w.UpdatedAt = ts
	return saveWallet(ctx, w)
}

// ============================================================
// 10. UnfreezeWallet
// ============================================================

// UnfreezeWallet reverses a freeze on a wallet. Org3MSP (CBOS) authority required.
func (s *SmartContract) UnfreezeWallet(ctx contractapi.TransactionContextInterface, walletID string) error {
	// L1: empty string validation
	if walletID == "" {
		return fmt.Errorf("walletID cannot be empty")
	}
	if err := requireCBOS(ctx); err != nil {
		return err
	}
	w, err := loadWallet(ctx, walletID)
	if err != nil {
		return err
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	w.Frozen = false
	w.UpdatedAt = ts
	return saveWallet(ctx, w)
}

// ============================================================
// 11. ClaimCBOSFees
// ============================================================

// ClaimCBOSFees sweeps all accumulated FeeEntry records and credits the CBOS wallet.
// Bug 2 fix: fees collected via individual FeeEntry composite keys (one per transaction)
// are aggregated here in a single atomic sweep, avoiding the MVCC hot-key problem that
// arises when all concurrent transfers write directly to WALLET:CBOS.
func (s *SmartContract) ClaimCBOSFees(ctx contractapi.TransactionContextInterface, cbosWalletID string) error {
	// L1: empty string validation
	if cbosWalletID == "" {
		return fmt.Errorf("cbosWalletID cannot be empty")
	}
	if err := requireCBOS(ctx); err != nil {
		return err
	}
	cbosWallet, err := loadWallet(ctx, cbosWalletID)
	if err != nil {
		return err
	}
	if cbosWallet.Role != RoleCBOS {
		return fmt.Errorf("wallet %s is not a CBOS wallet", cbosWalletID)
	}
	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(KeyPrefixFee, []string{})
	if err != nil {
		return fmt.Errorf("failed to query fee entries: %w", err)
	}
	defer iter.Close()

	var totalFees int64
	var feeKeys []string
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return fmt.Errorf("failed to iterate fee entries: %w", err)
		}
		var entry FeeEntry
		if err := json.Unmarshal(kv.Value, &entry); err != nil {
			return fmt.Errorf("failed to unmarshal fee entry: %w", err)
		}
		totalFees += entry.Amount
		feeKeys = append(feeKeys, kv.Key)
	}
	if totalFees == 0 {
		return nil // nothing to claim
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	cbosWallet.Balance += totalFees
	cbosWallet.UpdatedAt = ts
	if err := saveWallet(ctx, cbosWallet); err != nil {
		return err
	}
	// Delete all FeeEntry keys atomically within this single transaction proposal
	for _, k := range feeKeys {
		if err := ctx.GetStub().DelState(k); err != nil {
			return fmt.Errorf("failed to delete fee entry: %w", err)
		}
	}
	txID := ctx.GetStub().GetTxID()
	tx := &Transaction{
		Amount:       totalFees,
		DocType:      DocTypeTransaction,
		Fee:          0,
		From:         "FEE_POOL",
		Participants: []string{cbosWalletID},
		Timestamp:    ts,
		To:           cbosWalletID,
		TxID:         txID,
		TxType:       TxTypeTransfer,
	}
	return saveTx(ctx, tx)
}

// ============================================================
// 12. UpdateKYCTier
// ============================================================

// UpdateKYCTier updates the KYC compliance tier of a wallet.
// Prototype note: guarded by requireOrg1OrOrg2 because Org3MSP (CBOS) has no endorsing
// peer. In production this would be a CBOS-initiated action; the endorsement policy would
// be extended or a dedicated CBOS client application would co-sign via the orderer identity.
func (s *SmartContract) UpdateKYCTier(ctx contractapi.TransactionContextInterface, walletID string, newTier int) error {
	// L1: empty string validation
	if walletID == "" {
		return fmt.Errorf("walletID cannot be empty")
	}
	if newTier < 1 || newTier > 3 {
		return fmt.Errorf("invalid kycTier %d: must be 1, 2, or 3", newTier)
	}
	if err := requireOrg1OrOrg2(ctx); err != nil {
		return err
	}
	w, err := loadWallet(ctx, walletID)
	if err != nil {
		return err
	}
	ts, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}
	w.KYCTier = newTier
	w.UpdatedAt = ts
	return saveWallet(ctx, w)
}

// ============================================================
// 13. GetWallet
// ============================================================

// GetWallet returns the current state of a wallet.
func (s *SmartContract) GetWallet(ctx contractapi.TransactionContextInterface, walletID string) (*Wallet, error) {
	// L1: empty string validation
	if walletID == "" {
		return nil, fmt.Errorf("walletID cannot be empty")
	}
	return loadWallet(ctx, walletID)
}

// ============================================================
// 14. GetBalance
// ============================================================

// GetBalance returns the current token balance of a wallet.
func (s *SmartContract) GetBalance(ctx contractapi.TransactionContextInterface, walletID string) (int64, error) {
	// L1: empty string validation
	if walletID == "" {
		return 0, fmt.Errorf("walletID cannot be empty")
	}
	w, err := loadWallet(ctx, walletID)
	if err != nil {
		return 0, err
	}
	return w.Balance, nil
}

// ============================================================
// 15. GetTransaction
// ============================================================

// GetTransaction returns a transaction record by its ID.
func (s *SmartContract) GetTransaction(ctx contractapi.TransactionContextInterface, txID string) (*Transaction, error) {
	// L1: empty string validation
	if txID == "" {
		return nil, fmt.Errorf("txID cannot be empty")
	}
	data, err := ctx.GetStub().GetState(txKey(txID))
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction %s: %w", txID, err)
	}
	if data == nil {
		return nil, fmt.Errorf("transaction %s does not exist", txID)
	}
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	return &tx, nil
}

// ============================================================
// 16. GetBurnRequest
// ============================================================

// GetBurnRequest returns a burn request by the TxID of its InitiateBurn transaction.
func (s *SmartContract) GetBurnRequest(ctx contractapi.TransactionContextInterface, burnTxID string) (*BurnRequest, error) {
	// L1: empty string validation
	if burnTxID == "" {
		return nil, fmt.Errorf("burnTxID cannot be empty")
	}
	data, err := ctx.GetStub().GetState(burnKey(burnTxID))
	if err != nil {
		return nil, fmt.Errorf("failed to read burn request %s: %w", burnTxID, err)
	}
	if data == nil {
		return nil, fmt.Errorf("burn request %s does not exist", burnTxID)
	}
	var burn BurnRequest
	if err := json.Unmarshal(data, &burn); err != nil {
		return nil, fmt.Errorf("failed to unmarshal burn request: %w", err)
	}
	return &burn, nil
}

// ============================================================
// 17. GetTotalSupply
// ============================================================

// GetTotalSupply returns the total number of tokens currently in circulation.
// Incremented by MintTokens; decremented by ApproveBurn. Allows CBOS and auditors
// to verify the 1:1 deposit-to-token backing ratio at any point in time.
func (s *SmartContract) GetTotalSupply(ctx contractapi.TransactionContextInterface) (int64, error) {
	data, err := ctx.GetStub().GetState(KeyTotalSupply)
	if err != nil {
		return 0, fmt.Errorf("failed to read total supply: %w", err)
	}
	if data == nil {
		return 0, nil // no tokens minted yet
	}
	var supply int64
	if err := json.Unmarshal(data, &supply); err != nil {
		return 0, fmt.Errorf("failed to unmarshal total supply: %w", err)
	}
	return supply, nil
}

// ============================================================
// 18. GetWalletHistory
// ============================================================

// GetWalletHistory returns the complete modification history of a wallet.
// C3 fix: errors from GetHistoryForKey are now propagated (were previously silenced).
// Reads from immutable blockchain history, not CouchDB.
func (s *SmartContract) GetWalletHistory(ctx contractapi.TransactionContextInterface, walletID string) ([]WalletHistoryEntry, error) {
	// L1: empty string validation
	if walletID == "" {
		return nil, fmt.Errorf("walletID cannot be empty")
	}
	// C3 fix: return error instead of silencing it
	iter, err := ctx.GetStub().GetHistoryForKey(walletKey(walletID))
	if err != nil {
		return nil, fmt.Errorf("failed to get history for wallet %s: %w", walletID, err)
	}
	defer iter.Close()

	history := make([]WalletHistoryEntry, 0)
	for iter.HasNext() {
		response, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate wallet history: %w", err)
		}
		entry := WalletHistoryEntry{
			TxID:      response.TxId,
			Timestamp: response.Timestamp.AsTime().Format(time.RFC3339),
			IsDelete:  response.IsDelete,
		}
		if !response.IsDelete && len(response.Value) > 0 {
			var w Wallet
			if err := json.Unmarshal(response.Value, &w); err != nil {
				return nil, fmt.Errorf("failed to unmarshal wallet history record: %w", err)
			}
			entry.Record = &w
		}
		history = append(history, entry)
	}
	return history, nil
}

// ============================================================
// 19. GetTransactionsByWallet
// ============================================================

// GetTransactionsByWallet returns paginated transactions involving a wallet.
// Bug 4 fix: uses $elemMatch on the participants array field instead of $or,
// which cannot be combined with sort in CouchDB Mango.
func (s *SmartContract) GetTransactionsByWallet(ctx contractapi.TransactionContextInterface, walletID string, pageSize int32, bookmark string) (*PaginatedTransactionResult, error) {
	// L1: empty string validation
	if walletID == "" {
		return nil, fmt.Errorf("walletID cannot be empty")
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive")
	}
	queryString := fmt.Sprintf(
		`{"selector":{"docType":"transaction","participants":{"$elemMatch":{"$eq":"%s"}}},"sort":[{"docType":"desc"},{"participants":"desc"},{"timestamp":"desc"}],"use_index":["_design/indexTxParticipants","indexTxParticipants"]}`,
		walletID,
	)
	iter, metadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer iter.Close()

	records := make([]*Transaction, 0)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate transactions: %w", err)
		}
		var tx Transaction
		if err := json.Unmarshal(kv.Value, &tx); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
		}
		records = append(records, &tx)
	}
	return &PaginatedTransactionResult{
		Records:             records,
		FetchedRecordsCount: metadata.FetchedRecordsCount,
		Bookmark:            metadata.Bookmark,
	}, nil
}

// ============================================================
// 20. GetLargeTransactions
// ============================================================

// GetLargeTransactions returns paginated transactions exceeding a threshold amount.
// Org3MSP (CBOS) AML monitoring function. Uses indexTxAmount.
func (s *SmartContract) GetLargeTransactions(ctx contractapi.TransactionContextInterface, threshold int64, pageSize int32, bookmark string) (*PaginatedTransactionResult, error) {
	if err := requireCBOS(ctx); err != nil {
		return nil, err
	}
	if threshold < 0 {
		return nil, fmt.Errorf("threshold must be non-negative")
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive")
	}
	queryString := fmt.Sprintf(
		`{"selector":{"docType":"transaction","amount":{"$gte":%d}},"sort":[{"docType":"desc"},{"amount":"desc"},{"timestamp":"desc"}],"use_index":["_design/indexTxAmount","indexTxAmount"]}`,
		threshold,
	)
	iter, metadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to query large transactions: %w", err)
	}
	defer iter.Close()

	records := make([]*Transaction, 0)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate transactions: %w", err)
		}
		var tx Transaction
		if err := json.Unmarshal(kv.Value, &tx); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
		}
		records = append(records, &tx)
	}
	return &PaginatedTransactionResult{
		Records:             records,
		FetchedRecordsCount: metadata.FetchedRecordsCount,
		Bookmark:            metadata.Bookmark,
	}, nil
}

// ============================================================
// 21. GetPendingBurnRequests
// ============================================================

// GetPendingBurnRequests returns paginated PENDING burn requests for CBOS review.
// Sorted by initiatedAt ascending — oldest requests are reviewed first.
func (s *SmartContract) GetPendingBurnRequests(ctx contractapi.TransactionContextInterface, pageSize int32, bookmark string) (*PaginatedBurnResult, error) {
	if err := requireCBOS(ctx); err != nil {
		return nil, err
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive")
	}
	queryString := `{"selector":{"docType":"burnRequest","status":"PENDING"},"sort":[{"docType":"asc"},{"status":"asc"},{"initiatedAt":"asc"}],"use_index":["_design/indexBurnStatus","indexBurnStatus"]}`
	iter, metadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending burn requests: %w", err)
	}
	defer iter.Close()

	records := make([]*BurnRequest, 0)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate burn requests: %w", err)
		}
		var burn BurnRequest
		if err := json.Unmarshal(kv.Value, &burn); err != nil {
			return nil, fmt.Errorf("failed to unmarshal burn request: %w", err)
		}
		records = append(records, &burn)
	}
	return &PaginatedBurnResult{
		Records:             records,
		FetchedRecordsCount: metadata.FetchedRecordsCount,
		Bookmark:            metadata.Bookmark,
	}, nil
}

// ============================================================
// 22. GetWalletsByRole
// ============================================================

// GetWalletsByRole returns paginated wallets filtered by role.
// Uses indexWalletRole for efficient CouchDB queries.
func (s *SmartContract) GetWalletsByRole(ctx contractapi.TransactionContextInterface, role string, pageSize int32, bookmark string) (*PaginatedWalletResult, error) {
	// L1: empty string validation
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive")
	}
	queryString := fmt.Sprintf(
		`{"selector":{"docType":"wallet","role":"%s"},"sort":[{"docType":"asc"},{"role":"asc"}],"use_index":["_design/indexWalletRole","indexWalletRole"]}`,
		role,
	)
	iter, metadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallets by role: %w", err)
	}
	defer iter.Close()

	records := make([]*Wallet, 0)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate wallets: %w", err)
		}
		var w Wallet
		if err := json.Unmarshal(kv.Value, &w); err != nil {
			return nil, fmt.Errorf("failed to unmarshal wallet: %w", err)
		}
		records = append(records, &w)
	}
	return &PaginatedWalletResult{
		Records:             records,
		FetchedRecordsCount: metadata.FetchedRecordsCount,
		Bookmark:            metadata.Bookmark,
	}, nil
}

// ============================================================
// main
// ============================================================

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating diaspora chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting diaspora chaincode: %v", err)
	}
}
