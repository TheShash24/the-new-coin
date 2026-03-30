package handlers

import (
	"net/http"
	"strings"
)

// handleBurns dispatches:
//
//	POST /burns                → InitiateBurn
//	GET  /burns?pageSize=...   → GetPendingBurnRequests
func (s *Server) handleBurns(w http.ResponseWriter, r *http.Request) {
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	switch r.Method {
	case http.MethodPost:
		var body struct {
			VendorID string `json:"vendorId"`
			Amount   string `json:"amount"`
			BurnRef  string `json:"burnRef"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if body.VendorID == "" || body.Amount == "" || body.BurnRef == "" {
			writeError(w, http.StatusBadRequest, "vendorId, amount, and burnRef are required")
			return
		}
		result, ok := submit(w, conn.Contract(), "InitiateBurn", body.VendorID, body.Amount, body.BurnRef)
		if !ok {
			return
		}
		writeResult(w, result)

	case http.MethodGet:
		q := r.URL.Query()
		pageSize := q.Get("pageSize")
		bookmark := q.Get("bookmark")
		if pageSize == "" {
			pageSize = "10"
		}
		result, ok := evaluate(w, conn.Contract(), "GetPendingBurnRequests", pageSize, bookmark)
		if !ok {
			return
		}
		writeResult(w, result)

	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

// handleBurnsWithID dispatches:
//
//	GET  /burns/{burnTxID}          → GetBurnRequest
//	POST /burns/{burnTxID}/approve  → ApproveBurn
//	POST /burns/{burnTxID}/reject   → RejectBurn
func (s *Server) handleBurnsWithID(w http.ResponseWriter, r *http.Request) {
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0]="burns", parts[1]={burnTxID}, parts[2]=action (optional)
	if len(parts) < 2 || parts[1] == "" {
		writeError(w, http.StatusBadRequest, "burn transaction ID required")
		return
	}
	burnTxID := parts[1]

	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		result, ok := evaluate(w, conn.Contract(), "GetBurnRequest", burnTxID)
		if !ok {
			return
		}
		writeResult(w, result)
		return
	}

	switch parts[2] {
	case "approve":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		result, ok := submit(w, conn.Contract(), "ApproveBurn", burnTxID)
		if !ok {
			return
		}
		if len(result) == 0 {
			writeJSON(w, http.StatusOK, map[string]string{"result": "burn approved"})
			return
		}
		writeResult(w, result)

	case "reject":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		result, ok := submit(w, conn.Contract(), "RejectBurn", burnTxID)
		if !ok {
			return
		}
		if len(result) == 0 {
			writeJSON(w, http.StatusOK, map[string]string{"result": "burn rejected"})
			return
		}
		writeResult(w, result)

	default:
		writeError(w, http.StatusNotFound, "unknown action: "+parts[2])
	}
}

// POST /fees/claim
func (s *Server) handleClaimCBOSFees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	var body struct {
		CBOSWalletID string `json:"cbosWalletId"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.CBOSWalletID == "" {
		writeError(w, http.StatusBadRequest, "cbosWalletId is required")
		return
	}
	result, ok := submit(w, conn.Contract(), "ClaimCBOSFees", body.CBOSWalletID)
	if !ok {
		return
	}
	if len(result) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"result": "fees claimed"})
		return
	}
	writeResult(w, result)
}
