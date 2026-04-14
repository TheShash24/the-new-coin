package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================
// MockClientIdentity
// ============================================================

type MockClientIdentity struct {
	MSPID    string
	ClientID string
}

var _ cid.ClientIdentity = (*MockClientIdentity)(nil)

func (m *MockClientIdentity) GetID() (string, error)    { return m.ClientID, nil }
func (m *MockClientIdentity) GetMSPID() (string, error) { return m.MSPID, nil }
func (m *MockClientIdentity) GetAttributeValue(attrName string) (string, bool, error) {
	return "", false, nil
}
func (m *MockClientIdentity) AssertAttributeValue(attrName, attrValue string) error {
	return fmt.Errorf("attribute not found")
}
func (m *MockClientIdentity) GetX509Certificate() (*x509.Certificate, error) {
	return nil, fmt.Errorf("not implemented")
}

// ============================================================
// MockStateIterator
// ============================================================

type MockStateIterator struct {
	kvs    []*queryresult.KV
	cursor int
}

func (m *MockStateIterator) HasNext() bool { return m.cursor < len(m.kvs) }
func (m *MockStateIterator) Close() error  { return nil }
func (m *MockStateIterator) Next() (*queryresult.KV, error) {
	if !m.HasNext() {
		return nil, fmt.Errorf("no more results")
	}
	kv := m.kvs[m.cursor]
	m.cursor++
	return kv, nil
}

// ============================================================
// MockHistoryIterator
// ============================================================

type MockHistoryIterator struct{}

func (m *MockHistoryIterator) HasNext() bool                               { return false }
func (m *MockHistoryIterator) Close() error                                { return nil }
func (m *MockHistoryIterator) Next() (*queryresult.KeyModification, error) { return nil, nil }

// ============================================================
// MockStub
// ============================================================

type MockStub struct {
	TxID  string
	State map[string][]byte
}

var _ shim.ChaincodeStubInterface = (*MockStub)(nil)

func NewMockStub(txID string) *MockStub {
	return &MockStub{TxID: txID, State: make(map[string][]byte)}
}

func (m *MockStub) GetArgs() [][]byte                            { return nil }
func (m *MockStub) GetStringArgs() []string                      { return nil }
func (m *MockStub) GetFunctionAndParameters() (string, []string) { return "", nil }
func (m *MockStub) GetArgsSlice() ([]byte, error)                { return nil, nil }
func (m *MockStub) GetTxID() string                              { return m.TxID }
func (m *MockStub) GetChannelID() string                         { return "mychannel" }
func (m *MockStub) InvokeChaincode(name string, args [][]byte, channel string) *peer.Response {
	return nil
}
func (m *MockStub) GetState(key string) ([]byte, error) { return m.State[key], nil }
func (m *MockStub) PutState(key string, value []byte) error {
	m.State[key] = value
	return nil
}
func (m *MockStub) DelState(key string) error {
	delete(m.State, key)
	return nil
}
func (m *MockStub) SetStateValidationParameter(key string, ep []byte) error { return nil }
func (m *MockStub) GetStateValidationParameter(key string) ([]byte, error)  { return nil, nil }
func (m *MockStub) GetStateByRange(start, end string) (shim.StateQueryIteratorInterface, error) {
	return &MockStateIterator{}, nil
}
func (m *MockStub) GetStateByRangeWithPagination(start, end string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &MockStateIterator{}, &peer.QueryResponseMetadata{}, nil
}
func (m *MockStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	key := "\x00" + objectType + "\x00"
	for _, attr := range attributes {
		key += attr + "\x00"
	}
	return key, nil
}
func (m *MockStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return "", nil, nil
}
func (m *MockStub) GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	prefix, _ := m.CreateCompositeKey(objectType, keys)
	var kvs []*queryresult.KV
	for k, v := range m.State {
		if strings.HasPrefix(k, prefix) {
			kvs = append(kvs, &queryresult.KV{Key: k, Value: v})
		}
	}
	return &MockStateIterator{kvs: kvs}, nil
}
func (m *MockStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &MockStateIterator{}, &peer.QueryResponseMetadata{}, nil
}
func (m *MockStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {
	return &MockStateIterator{}, nil
}
func (m *MockStub) GetQueryResultWithPagination(query string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &MockStateIterator{}, &peer.QueryResponseMetadata{}, nil
}
func (m *MockStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return &MockHistoryIterator{}, nil
}
func (m *MockStub) GetPrivateData(collection, key string) ([]byte, error)     { return nil, nil }
func (m *MockStub) GetPrivateDataHash(collection, key string) ([]byte, error) { return nil, nil }
func (m *MockStub) PutPrivateData(collection, key string, value []byte) error { return nil }
func (m *MockStub) DelPrivateData(collection, key string) error               { return nil }
func (m *MockStub) PurgePrivateData(collection, key string) error             { return nil }
func (m *MockStub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	return nil
}
func (m *MockStub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateDataByRange(collection, start, end string) (shim.StateQueryIteratorInterface, error) {
	return &MockStateIterator{}, nil
}
func (m *MockStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return &MockStateIterator{}, nil
}
func (m *MockStub) GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error) {
	return &MockStateIterator{}, nil
}
func (m *MockStub) GetCreator() ([]byte, error)              { return nil, nil }
func (m *MockStub) GetTransient() (map[string][]byte, error) { return nil, nil }
func (m *MockStub) GetBinding() ([]byte, error)              { return nil, nil }
func (m *MockStub) GetDecorations() map[string][]byte        { return nil }
func (m *MockStub) GetSignedProposal() (*peer.SignedProposal, error) {
	return nil, nil
}
func (m *MockStub) GetTxTimestamp() (*timestamppb.Timestamp, error) {
	return timestamppb.Now(), nil
}
func (m *MockStub) SetEvent(name string, payload []byte) error { return nil }

// ============================================================
// MockTransactionContext
// ============================================================

type MockTransactionContext struct {
	Stub     *MockStub
	Identity cid.ClientIdentity
}

func (m *MockTransactionContext) GetStub() shim.ChaincodeStubInterface { return m.Stub }
func (m *MockTransactionContext) GetClientIdentity() cid.ClientIdentity { return m.Identity }

// ============================================================
// Test helpers
// ============================================================

func newCtx(mspID, clientID, txID string) *MockTransactionContext {
	return &MockTransactionContext{
		Stub:     NewMockStub(txID),
		Identity: &MockClientIdentity{MSPID: mspID, ClientID: clientID},
	}
}

func mustPutWallet(stub *MockStub, w *Wallet) {
	data, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}
	if err := stub.PutState(walletKey(w.ID), data); err != nil {
		panic(err)
	}
}

func mustGetWallet(t *testing.T, stub *MockStub, id string) *Wallet {
	t.Helper()
	data, err := stub.GetState(walletKey(id))
	if err != nil || data == nil {
		t.Fatalf("wallet %s not found in state", id)
	}
	var w Wallet
	if err := json.Unmarshal(data, &w); err != nil {
		t.Fatalf("unmarshal wallet %s: %v", id, err)
	}
	return &w
}

func seedTotalSupply(stub *MockStub, amount int64) {
	data, _ := json.Marshal(amount)
	_ = stub.PutState(KeyTotalSupply, data)
}

// ============================================================
// RegisterWallet tests
// ============================================================

func TestRegisterWallet_Valid(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	err := sc.RegisterWallet(ctx, "w1", RoleDiaspora, 2)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	w := mustGetWallet(t, ctx.Stub, "w1")
	if w.Role != RoleDiaspora {
		t.Errorf("expected role %s, got %s", RoleDiaspora, w.Role)
	}
	if w.ID != "w1" {
		t.Errorf("expected id w1, got %s", w.ID)
	}
	if w.Owner != "user1" {
		t.Errorf("expected owner user1, got %s", w.Owner)
	}
}

func TestRegisterWallet_WrongMSP(t *testing.T) {
	ctx := newCtx(MSPOrg3, "user1", "tx1")
	sc := new(SmartContract)
	err := sc.RegisterWallet(ctx, "w1", RoleDiaspora, 2)
	if err == nil {
		t.Fatal("expected error for Org3MSP caller, got nil")
	}
}

func TestRegisterWallet_CBOSRole(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	err := sc.RegisterWallet(ctx, "w1", RoleCBOS, 2)
	if err == nil {
		t.Fatal("expected error for CBOS role, got nil")
	}
}

func TestRegisterWallet_Duplicate(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	if err := sc.RegisterWallet(ctx, "w1", RoleDiaspora, 2); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	err := sc.RegisterWallet(ctx, "w1", RoleDiaspora, 2)
	if err == nil {
		t.Fatal("expected error for duplicate wallet, got nil")
	}
}

// ============================================================
// MintTokens tests
// ============================================================

func TestMintTokens_Valid(t *testing.T) {
	ctx := newCtx(MSPOrg1, "bank1", "tx-mint")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "w1", Role: RoleDiaspora, Owner: "bank1",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.MintTokens(ctx, "w1", 1000, "DEP-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	w := mustGetWallet(t, ctx.Stub, "w1")
	if w.Balance != 1000 {
		t.Errorf("expected balance 1000, got %d", w.Balance)
	}
}

func TestMintTokens_EmptyDepositRef(t *testing.T) {
	ctx := newCtx(MSPOrg1, "bank1", "tx1")
	sc := new(SmartContract)
	err := sc.MintTokens(ctx, "w1", 1000, "")
	if err == nil {
		t.Fatal("expected error for empty depositRef, got nil")
	}
}

func TestMintTokens_WrongMSP(t *testing.T) {
	ctx := newCtx(MSPOrg2, "bank2", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "w1", Role: RoleDiaspora, Owner: "bank2",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.MintTokens(ctx, "w1", 1000, "DEP-001")
	if err == nil {
		t.Fatal("expected error for Org2MSP caller, got nil")
	}
}

func TestMintTokens_DuplicateDepositRef(t *testing.T) {
	ctx := newCtx(MSPOrg1, "bank1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "w1", Role: RoleDiaspora, Owner: "bank1",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	if err := sc.MintTokens(ctx, "w1", 1000, "DEP-001"); err != nil {
		t.Fatalf("first mint failed: %v", err)
	}
	err := sc.MintTokens(ctx, "w1", 1000, "DEP-001")
	if err == nil {
		t.Fatal("expected error for duplicate depositRef, got nil")
	}
}

// ============================================================
// Transfer tests
// ============================================================

func TestTransfer_Valid(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx-transfer")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora1", Role: RoleDiaspora, Owner: "user1",
		Balance: 2000, KYCTier: 3, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "relative1", Role: RoleRelative, Owner: "user2",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.Transfer(ctx, "diaspora1", "relative1", 1000)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	fee := calculateFee(1000)
	sender := mustGetWallet(t, ctx.Stub, "diaspora1")
	recipient := mustGetWallet(t, ctx.Stub, "relative1")
	if sender.Balance != 2000-1000-fee {
		t.Errorf("expected sender balance %d, got %d", 2000-1000-fee, sender.Balance)
	}
	if recipient.Balance != 1000 {
		t.Errorf("expected recipient balance 1000, got %d", recipient.Balance)
	}
}

func TestTransfer_FrozenSender(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora1", Role: RoleDiaspora, Owner: "user1",
		Balance: 2000, KYCTier: 3, Frozen: true, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "relative1", Role: RoleRelative, Owner: "user2",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.Transfer(ctx, "diaspora1", "relative1", 100)
	if err == nil {
		t.Fatal("expected error for frozen sender, got nil")
	}
}

func TestTransfer_FrozenRecipient(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora1", Role: RoleDiaspora, Owner: "user1",
		Balance: 2000, KYCTier: 3, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "relative1", Role: RoleRelative, Owner: "user2",
		Balance: 0, KYCTier: 3, Frozen: true, DocType: DocTypeWallet,
	})
	err := sc.Transfer(ctx, "diaspora1", "relative1", 100)
	if err == nil {
		t.Fatal("expected error for frozen recipient, got nil")
	}
}

func TestTransfer_WrongSenderRole(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	// Sender has RELATIVE role instead of DIASPORA
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "rel-as-sender", Role: RoleRelative, Owner: "user1",
		Balance: 2000, KYCTier: 3, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "relative1", Role: RoleRelative, Owner: "user2",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.Transfer(ctx, "rel-as-sender", "relative1", 100)
	if err == nil {
		t.Fatal("expected error for wrong sender role, got nil")
	}
}

func TestTransfer_WrongRecipientRole(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora1", Role: RoleDiaspora, Owner: "user1",
		Balance: 2000, KYCTier: 3, DocType: DocTypeWallet,
	})
	// Recipient has DIASPORA role instead of RELATIVE
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora2", Role: RoleDiaspora, Owner: "user2",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.Transfer(ctx, "diaspora1", "diaspora2", 100)
	if err == nil {
		t.Fatal("expected error for wrong recipient role, got nil")
	}
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora1", Role: RoleDiaspora, Owner: "user1",
		Balance: 50, KYCTier: 3, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "relative1", Role: RoleRelative, Owner: "user2",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.Transfer(ctx, "diaspora1", "relative1", 500)
	if err == nil {
		t.Fatal("expected error for insufficient balance, got nil")
	}
}

// ============================================================
// PayVendor tests
// ============================================================

func TestPayVendor_Valid(t *testing.T) {
	ctx := newCtx(MSPOrg2, "user2", "tx-pay")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "rel1", Role: RoleRelative, Owner: "user2",
		Balance: 2000, KYCTier: 3, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "vendor1", Role: RoleVendor, Owner: "vendor_owner",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.PayVendor(ctx, "rel1", "vendor1", 1000)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	fee := calculateFee(1000)
	payer := mustGetWallet(t, ctx.Stub, "rel1")
	vendor := mustGetWallet(t, ctx.Stub, "vendor1")
	if payer.Balance != 2000-1000-fee {
		t.Errorf("expected payer balance %d, got %d", 2000-1000-fee, payer.Balance)
	}
	if vendor.Balance != 1000 {
		t.Errorf("expected vendor balance 1000, got %d", vendor.Balance)
	}
}

func TestPayVendor_WrongPayerRole(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	// Payer has DIASPORA role instead of RELATIVE
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "diaspora1", Role: RoleDiaspora, Owner: "user1",
		Balance: 2000, KYCTier: 3, DocType: DocTypeWallet,
	})
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "vendor1", Role: RoleVendor, Owner: "vendor_owner",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.PayVendor(ctx, "diaspora1", "vendor1", 100)
	if err == nil {
		t.Fatal("expected error for wrong payer role, got nil")
	}
}

// ============================================================
// InitiateBurn tests
// ============================================================

func TestInitiateBurn_Valid(t *testing.T) {
	ctx := newCtx(MSPOrg2, "vendor_user", "tx-initiate-burn")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "vendor1", Role: RoleVendor, Owner: "vendor_user",
		Balance: 500, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.InitiateBurn(ctx, "vendor1", 300, "BURN-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	w := mustGetWallet(t, ctx.Stub, "vendor1")
	if w.Balance != 200 {
		t.Errorf("expected vendor balance 200 after escrow, got %d", w.Balance)
	}
	burnData, _ := ctx.Stub.GetState(burnKey(ctx.Stub.GetTxID()))
	if burnData == nil {
		t.Fatal("expected burn request to be created")
	}
	var burn BurnRequest
	if err := json.Unmarshal(burnData, &burn); err != nil {
		t.Fatalf("unmarshal burn request: %v", err)
	}
	if burn.Status != BurnStatusPending {
		t.Errorf("expected status PENDING, got %s", burn.Status)
	}
}

// ============================================================
// ApproveBurn tests
// ============================================================

func TestApproveBurn_FromOrg3(t *testing.T) {
	ctx := newCtx(MSPOrg3, "cbos_user", "tx-approve")
	sc := new(SmartContract)
	burn := &BurnRequest{
		Amount:   300,
		BurnRef:  "BURN-001",
		DocType:  DocTypeBurnRequest,
		ID:       "burn-tx-001",
		Status:   BurnStatusPending,
		VendorID: "vendor1",
	}
	burnData, _ := json.Marshal(burn)
	ctx.Stub.PutState(burnKey("burn-tx-001"), burnData)
	seedTotalSupply(ctx.Stub, 300)

	err := sc.ApproveBurn(ctx, "burn-tx-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	burnDataAfter, _ := ctx.Stub.GetState(burnKey("burn-tx-001"))
	var burnAfter BurnRequest
	json.Unmarshal(burnDataAfter, &burnAfter)
	if burnAfter.Status != BurnStatusApproved {
		t.Errorf("expected status APPROVED, got %s", burnAfter.Status)
	}
}

func TestApproveBurn_FromOrg1(t *testing.T) {
	ctx := newCtx(MSPOrg1, "bank1", "tx1")
	sc := new(SmartContract)
	burn := &BurnRequest{
		Amount:   300,
		BurnRef:  "BURN-001",
		DocType:  DocTypeBurnRequest,
		ID:       "burn-tx-001",
		Status:   BurnStatusPending,
		VendorID: "vendor1",
	}
	burnData, _ := json.Marshal(burn)
	ctx.Stub.PutState(burnKey("burn-tx-001"), burnData)

	err := sc.ApproveBurn(ctx, "burn-tx-001")
	if err == nil {
		t.Fatal("expected error for Org1MSP caller, got nil")
	}
}

// ============================================================
// RejectBurn test
// ============================================================

func TestRejectBurn(t *testing.T) {
	ctx := newCtx(MSPOrg3, "cbos_user", "tx-reject")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "vendor1", Role: RoleVendor, Owner: "vendor_user",
		Balance: 200, KYCTier: 3, DocType: DocTypeWallet,
	})
	burn := &BurnRequest{
		Amount:   300,
		BurnRef:  "BURN-001",
		DocType:  DocTypeBurnRequest,
		ID:       "burn-tx-001",
		Status:   BurnStatusPending,
		VendorID: "vendor1",
	}
	burnData, _ := json.Marshal(burn)
	ctx.Stub.PutState(burnKey("burn-tx-001"), burnData)

	err := sc.RejectBurn(ctx, "burn-tx-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	w := mustGetWallet(t, ctx.Stub, "vendor1")
	if w.Balance != 500 {
		t.Errorf("expected vendor balance 500 after refund, got %d", w.Balance)
	}
	burnDataAfter, _ := ctx.Stub.GetState(burnKey("burn-tx-001"))
	var burnAfter BurnRequest
	json.Unmarshal(burnDataAfter, &burnAfter)
	if burnAfter.Status != BurnStatusRejected {
		t.Errorf("expected status REJECTED, got %s", burnAfter.Status)
	}
}

// ============================================================
// FreezeWallet tests
// ============================================================

func TestFreezeWallet_FromOrg3(t *testing.T) {
	ctx := newCtx(MSPOrg3, "cbos_user", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "w1", Role: RoleDiaspora, Owner: "user1",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.FreezeWallet(ctx, "w1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	w := mustGetWallet(t, ctx.Stub, "w1")
	if !w.Frozen {
		t.Error("expected wallet to be frozen")
	}
}

func TestFreezeWallet_CBOSWallet(t *testing.T) {
	ctx := newCtx(MSPOrg3, "cbos_user", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "cbos1", Role: RoleCBOS, Owner: "CBOS_SYSTEM",
		Balance: 0, KYCTier: 3, DocType: DocTypeWallet,
	})
	err := sc.FreezeWallet(ctx, "cbos1")
	if err == nil {
		t.Fatal("expected error for freezing CBOS wallet, got nil")
	}
}

// ============================================================
// ClaimCBOSFees test
// ============================================================

func TestClaimCBOSFees(t *testing.T) {
	ctx := newCtx(MSPOrg3, "cbos_user", "tx-claim")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: CBOSWalletID, Role: RoleCBOS, Owner: "CBOS_SYSTEM",
		Balance: 100, KYCTier: 3, DocType: DocTypeWallet,
	})
	feeKey1, _ := ctx.Stub.CreateCompositeKey(KeyPrefixFee, []string{"tx001"})
	feeKey2, _ := ctx.Stub.CreateCompositeKey(KeyPrefixFee, []string{"tx002"})
	fee1 := FeeEntry{Amount: 10, DocType: DocTypeFee, Source: "w1", Timestamp: "2025-01-01T00:00:00Z", TxID: "tx001"}
	fee2 := FeeEntry{Amount: 15, DocType: DocTypeFee, Source: "w2", Timestamp: "2025-01-01T00:00:00Z", TxID: "tx002"}
	d1, _ := json.Marshal(fee1)
	d2, _ := json.Marshal(fee2)
	ctx.Stub.PutState(feeKey1, d1)
	ctx.Stub.PutState(feeKey2, d2)

	err := sc.ClaimCBOSFees(ctx, CBOSWalletID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	w := mustGetWallet(t, ctx.Stub, CBOSWalletID)
	if w.Balance != 125 {
		t.Errorf("expected CBOS balance 125, got %d", w.Balance)
	}
	d, _ := ctx.Stub.GetState(feeKey1)
	if d != nil {
		t.Error("expected fee key 1 to be deleted after claim")
	}
}

// ============================================================
// GetWallet test
// ============================================================

func TestGetWallet(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "w1", Role: RoleDiaspora, Owner: "user1",
		Balance: 750, KYCTier: 2, DocType: DocTypeWallet,
	})
	w, err := sc.GetWallet(ctx, "w1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if w.ID != "w1" {
		t.Errorf("expected id w1, got %s", w.ID)
	}
	if w.Balance != 750 {
		t.Errorf("expected balance 750, got %d", w.Balance)
	}
	if w.Role != RoleDiaspora {
		t.Errorf("expected role %s, got %s", RoleDiaspora, w.Role)
	}
}

// ============================================================
// GetBalance test
// ============================================================

func TestGetBalance(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	mustPutWallet(ctx.Stub, &Wallet{
		ID: "w1", Role: RoleDiaspora, Owner: "user1",
		Balance: 750, KYCTier: 2, DocType: DocTypeWallet,
	})
	balance, err := sc.GetBalance(ctx, "w1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if balance != 750 {
		t.Errorf("expected balance 750, got %d", balance)
	}
}

// ============================================================
// Empty string inputs test
// ============================================================

func TestEmptyStringInputs(t *testing.T) {
	ctx := newCtx(MSPOrg1, "user1", "tx1")
	sc := new(SmartContract)
	err := sc.RegisterWallet(ctx, "", RoleDiaspora, 2)
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
}
