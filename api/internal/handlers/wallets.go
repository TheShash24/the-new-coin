package handlers

import (
	"net/http"
	"strings"

	gw "github.com/hyperledger/fabric-samples/diaspora-api/internal/gateway"
)

// handleWallets dispatches GET /wallets (list by role) and POST /wallets (register).
func (s *Server) handleWallets(w http.ResponseWriter, r *http.Request) {
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.registerWallet(w, r, conn)
	case http.MethodGet:
		s.getWalletsByRole(w, r, conn)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

// handleWalletsWithID dispatches routes under /wallets/{id}[/sub-resource].
//
//	GET  /wallets/{id}           → GetWallet
//	GET  /wallets/{id}/balance   → GetBalance
//	GET  /wallets/{id}/history   → GetWalletHistory
//	POST /wallets/{id}/freeze    → FreezeWallet
//	POST /wallets/{id}/unfreeze  → UnfreezeWallet
//	PUT  /wallets/{id}/kyc       → UpdateKYCTier
func (s *Server) handleWalletsWithID(w http.ResponseWriter, r *http.Request) {
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0]="wallets", parts[1]={id}, parts[2]=sub (optional)
	if len(parts) < 2 || parts[1] == "" {
		writeError(w, http.StatusBadRequest, "wallet ID required")
		return
	}
	walletID := parts[1]

	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		result, ok := evaluate(w, conn.Contract(), "GetWallet", walletID)
		if !ok {
			return
		}
		writeResult(w, result)
		return
	}

	switch parts[2] {
	case "balance":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		result, ok := evaluate(w, conn.Contract(), "GetBalance", walletID)
		if !ok {
			return
		}
		writeResult(w, result)

	case "history":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		result, ok := evaluate(w, conn.Contract(), "GetWalletHistory", walletID)
		if !ok {
			return
		}
		writeResult(w, result)

	case "freeze":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		result, ok := submit(w, conn.Contract(), "FreezeWallet", walletID)
		if !ok {
			return
		}
		if len(result) == 0 {
			writeJSON(w, http.StatusOK, map[string]string{"result": "wallet frozen"})
			return
		}
		writeResult(w, result)

	case "unfreeze":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		result, ok := submit(w, conn.Contract(), "UnfreezeWallet", walletID)
		if !ok {
			return
		}
		if len(result) == 0 {
			writeJSON(w, http.StatusOK, map[string]string{"result": "wallet unfrozen"})
			return
		}
		writeResult(w, result)

	case "kyc":
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "PUT required")
			return
		}
		var body struct {
			NewTier string `json:"newTier"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if body.NewTier == "" {
			writeError(w, http.StatusBadRequest, "newTier is required")
			return
		}
		result, ok := submit(w, conn.Contract(), "UpdateKYCTier", walletID, body.NewTier)
		if !ok {
			return
		}
		if len(result) == 0 {
			writeJSON(w, http.StatusOK, map[string]string{"result": "KYC tier updated"})
			return
		}
		writeResult(w, result)

	default:
		writeError(w, http.StatusNotFound, "unknown sub-resource: "+parts[2])
	}
}

func (s *Server) registerWallet(w http.ResponseWriter, r *http.Request, conn *gw.OrgConnection) {
	var body struct {
		ID      string `json:"id"`
		Role    string `json:"role"`
		KYCTier string `json:"kycTier"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.ID == "" || body.Role == "" || body.KYCTier == "" {
		writeError(w, http.StatusBadRequest, "id, role, and kycTier are required")
		return
	}
	result, ok := submit(w, conn.Contract(), "RegisterWallet", body.ID, body.Role, body.KYCTier)
	if !ok {
		return
	}
	if len(result) == 0 {
		writeJSON(w, http.StatusCreated, map[string]string{"result": "wallet registered"})
		return
	}
	writeResult(w, result)
}

func (s *Server) getWalletsByRole(w http.ResponseWriter, r *http.Request, conn *gw.OrgConnection) {
	q := r.URL.Query()
	role := q.Get("role")
	pageSize := q.Get("pageSize")
	bookmark := q.Get("bookmark")
	if role == "" {
		writeError(w, http.StatusBadRequest, "role query parameter is required")
		return
	}
	if pageSize == "" {
		pageSize = "10"
	}
	result, ok := evaluate(w, conn.Contract(), "GetWalletsByRole", role, pageSize, bookmark)
	if !ok {
		return
	}
	writeResult(w, result)
}
