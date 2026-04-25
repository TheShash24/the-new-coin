package handlers

import (
	"net/http"
)

// POST /tokens/mint
func (s *Server) handleMintTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	var body struct {
		WalletID         string `json:"walletId"`
		Amount           string `json:"amount"`
		DepositRef       string `json:"depositRef"`
		OriginalCurrency string `json:"originalCurrency"`
		OriginalAmount   string `json:"originalAmount"`
		ExchangeRate     string `json:"exchangeRate"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.WalletID == "" || body.Amount == "" || body.DepositRef == "" {
		writeError(w, http.StatusBadRequest, "walletId, amount, and depositRef are required")
		return
	}
	result, ok := submit(w, conn.Contract(), "MintTokens", body.WalletID, body.Amount, body.DepositRef, body.OriginalCurrency, body.OriginalAmount, body.ExchangeRate)
	if !ok {
		return
	}
	if len(result) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"result": "tokens minted"})
		return
	}
	writeResult(w, result)
}

// POST /tokens/transfer
func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	var body struct {
		FromID string `json:"fromId"`
		ToID   string `json:"toId"`
		Amount string `json:"amount"`
		Memo   string `json:"memo"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.FromID == "" || body.ToID == "" || body.Amount == "" {
		writeError(w, http.StatusBadRequest, "fromId, toId, and amount are required")
		return
	}
	result, ok := submit(w, conn.Contract(), "Transfer", body.FromID, body.ToID, body.Amount, body.Memo)
	if !ok {
		return
	}
	writeResult(w, result)
}

// POST /tokens/pay
func (s *Server) handlePayVendor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	var body struct {
		FromID string `json:"fromId"`
		ToID   string `json:"toId"`
		Amount string `json:"amount"`
		Memo   string `json:"memo"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.FromID == "" || body.ToID == "" || body.Amount == "" {
		writeError(w, http.StatusBadRequest, "fromId, toId, and amount are required")
		return
	}
	result, ok := submit(w, conn.Contract(), "PayVendor", body.FromID, body.ToID, body.Amount, body.Memo)
	if !ok {
		return
	}
	writeResult(w, result)
}

// GET /tokens/supply
func (s *Server) handleGetTotalSupply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	result, ok := evaluate(w, conn.Contract(), "GetTotalSupply")
	if !ok {
		return
	}
	writeResult(w, result)
}

// GET /tokens/transactions?walletId=...&pageSize=...&bookmark=...
func (s *Server) handleGetTransactionsByWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	q := r.URL.Query()
	walletID := q.Get("walletId")
	pageSize := q.Get("pageSize")
	bookmark := q.Get("bookmark")
	if walletID == "" {
		writeError(w, http.StatusBadRequest, "walletId query parameter is required")
		return
	}
	if pageSize == "" {
		pageSize = "10"
	}
	result, ok := evaluate(w, conn.Contract(), "GetTransactionsByWallet", walletID, pageSize, bookmark)
	if !ok {
		return
	}
	writeResult(w, result)
}

// GET /tokens/large-transactions?threshold=...&pageSize=...&bookmark=...
func (s *Server) handleGetLargeTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	q := r.URL.Query()
	threshold := q.Get("threshold")
	pageSize := q.Get("pageSize")
	bookmark := q.Get("bookmark")
	if threshold == "" {
		writeError(w, http.StatusBadRequest, "threshold query parameter is required")
		return
	}
	if pageSize == "" {
		pageSize = "10"
	}
	result, ok := evaluate(w, conn.Contract(), "GetLargeTransactions", threshold, pageSize, bookmark)
	if !ok {
		return
	}
	writeResult(w, result)
}

// GET /transactions/{txID}
func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	txID := pathSegment(r, 1) // /transactions/{txID}
	if txID == "" {
		writeError(w, http.StatusBadRequest, "transaction ID required")
		return
	}
	result, ok := evaluate(w, conn.Contract(), "GetTransaction", txID)
	if !ok {
		return
	}
	writeResult(w, result)
}
