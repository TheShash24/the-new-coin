package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	gw "github.com/hyperledger/fabric-samples/diaspora-api/internal/gateway"
)

// Server holds all org connections and exposes HTTP handlers.
type Server struct {
	orgs map[string]*gw.OrgConnection
}

// NewServer creates a Server with the provided org connections.
func NewServer(orgs map[string]*gw.OrgConnection) *Server {
	return &Server{orgs: orgs}
}

// Routes registers all endpoints and returns the root mux.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Ledger init
	mux.HandleFunc("/ledger/init", s.handleInitLedger)

	// Wallet endpoints
	mux.HandleFunc("/wallets", s.handleWallets)
	mux.HandleFunc("/wallets/", s.handleWalletsWithID)

	// Token endpoints
	mux.HandleFunc("/tokens/mint", s.handleMintTokens)
	mux.HandleFunc("/tokens/transfer", s.handleTransfer)
	mux.HandleFunc("/tokens/pay", s.handlePayVendor)
	mux.HandleFunc("/tokens/supply", s.handleGetTotalSupply)
	mux.HandleFunc("/tokens/transactions", s.handleGetTransactionsByWallet)
	mux.HandleFunc("/tokens/large-transactions", s.handleGetLargeTransactions)

	// Transaction lookup
	mux.HandleFunc("/transactions/", s.handleGetTransaction)

	// Burn endpoints
	mux.HandleFunc("/burns", s.handleBurns)
	mux.HandleFunc("/burns/", s.handleBurnsWithID)
	mux.HandleFunc("/fees/claim", s.handleClaimCBOSFees)

	return corsMiddleware(loggingMiddleware(mux))
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Org-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s org=%s status=%d", r.Method, r.URL.Path, r.Header.Get("X-Org-ID"), rw.status)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// orgFromHeader resolves the OrgConnection from the X-Org-ID header.
// Returns nil and writes a 400 response if the header is missing or invalid.
func (s *Server) orgFromHeader(w http.ResponseWriter, r *http.Request) *gw.OrgConnection {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		writeError(w, http.StatusBadRequest, "X-Org-ID header is required")
		return nil
	}
	conn, ok := s.orgs[orgID]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown X-Org-ID: %s (valid: Org1, Org2, Org3)", orgID))
		return nil
	}
	return conn
}

// submit calls SubmitTransaction and maps Gateway errors to HTTP status codes.
func submit(w http.ResponseWriter, contract *client.Contract, fn string, args ...string) ([]byte, bool) {
	result, err := contract.SubmitTransaction(fn, args...)
	if err != nil {
		writeGatewayError(w, err)
		return nil, false
	}
	return result, true
}

// evaluate calls EvaluateTransaction and maps Gateway errors to HTTP status codes.
func evaluate(w http.ResponseWriter, contract *client.Contract, fn string, args ...string) ([]byte, bool) {
	result, err := contract.EvaluateTransaction(fn, args...)
	if err != nil {
		writeGatewayError(w, err)
		return nil, false
	}
	return result, true
}

// writeGatewayError maps Fabric Gateway SDK errors to appropriate HTTP status codes.
func writeGatewayError(w http.ResponseWriter, err error) {
	var endorseErr *client.EndorseError
	var submitErr *client.SubmitError
	var commitErr *client.CommitStatusError
	var txErr *client.CommitError

	switch {
	case errors.As(err, &endorseErr):
		writeError(w, http.StatusBadRequest, endorseErr.Error())
	case errors.As(err, &submitErr):
		writeError(w, http.StatusBadRequest, submitErr.Error())
	case errors.As(err, &commitErr):
		writeError(w, http.StatusInternalServerError, commitErr.Error())
	case errors.As(err, &txErr):
		writeError(w, http.StatusBadRequest, txErr.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// rawOrObject writes a result: if it is valid JSON it is forwarded as-is,
// otherwise it is wrapped in {"result": "<string>"}.
func writeResult(w http.ResponseWriter, data []byte) {
	if json.Valid(data) && len(data) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": strings.TrimSpace(string(data))})
}

// decodeBody decodes a JSON request body into v.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// pathSegment returns the URL path segment at position n (0-indexed, split by "/").
// e.g. for "/wallets/abc123" → pathSegment(r, 2) == "abc123"
func pathSegment(r *http.Request, n int) string {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if n < len(parts) {
		return parts[n]
	}
	return ""
}

// ---------------------------------------------------------------------------
// InitLedger
// ---------------------------------------------------------------------------

func (s *Server) handleInitLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	conn := s.orgFromHeader(w, r)
	if conn == nil {
		return
	}
	result, ok := submit(w, conn.Contract(), "InitLedger")
	if !ok {
		return
	}
	if len(result) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"result": "ledger initialised"})
		return
	}
	writeResult(w, result)
}
