package gateway

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// OrgConfig holds the crypto material paths and connection details for one org.
type OrgConfig struct {
	OrgName      string // "Org1", "Org2", "Org3"
	MSPID        string
	CertPath     string // path to signcerts/cert.pem
	KeyPath      string // path to keystore/ directory
	TLSCertPath  string // path to peers/peer0.orgN.example.com/tls/ca.crt
	PeerEndpoint string // e.g. "dns:///localhost:7051"
	GatewayPeer  string // TLS server name, e.g. "peer0.org1.example.com"
}

// OrgConnection holds a persistent Gateway + Contract for one org.
type OrgConnection struct {
	MSPID    string
	gateway  *client.Gateway
	contract *client.Contract
}

// Contract returns the cached chaincode contract handle.
func (o *OrgConnection) Contract() *client.Contract {
	return o.contract
}

// Close releases the underlying gRPC connection.
func (o *OrgConnection) Close() {
	o.gateway.Close()
}

// Initialize connects to the Fabric peer for the given org and returns an OrgConnection.
func Initialize(cfg OrgConfig) (*OrgConnection, error) {
	// Load TLS certificate for the peer.
	tlsCertPEM, err := os.ReadFile(cfg.TLSCertPath)
	if err != nil {
		return nil, fmt.Errorf("read TLS cert %s: %w", cfg.TLSCertPath, err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(tlsCertPEM) {
		return nil, fmt.Errorf("failed to add TLS cert to pool for %s", cfg.OrgName)
	}
	tlsCreds := credentials.NewClientTLSFromCert(certPool, cfg.GatewayPeer)

	conn, err := grpc.NewClient(cfg.PeerEndpoint, grpc.WithTransportCredentials(tlsCreds))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", cfg.PeerEndpoint, err)
	}

	// Load signing identity certificate.
	certPEM, err := os.ReadFile(cfg.CertPath)
	if err != nil {
		return nil, fmt.Errorf("read cert %s: %w", cfg.CertPath, err)
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse cert for %s: %w", cfg.OrgName, err)
	}
	id, err := identity.NewX509Identity(cfg.MSPID, cert)
	if err != nil {
		return nil, fmt.Errorf("create identity for %s: %w", cfg.OrgName, err)
	}

	// Load private key from keystore directory (first file found).
	keyPEM, err := firstFileInDir(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read keystore %s: %w", cfg.KeyPath, err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key for %s: %w", cfg.OrgName, err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, fmt.Errorf("create signer for %s: %w", cfg.OrgName, err)
	}

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("gateway connect for %s: %w", cfg.OrgName, err)
	}

	contract := gw.GetNetwork("mychannel").GetContract("diaspora")

	return &OrgConnection{
		MSPID:    cfg.MSPID,
		gateway:  gw,
		contract: contract,
	}, nil
}

// firstFileInDir reads the contents of the first file found in dir.
func firstFileInDir(dir string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			return os.ReadFile(filepath.Join(dir, e.Name()))
		}
	}
	return nil, fmt.Errorf("no files found in %s", dir)
}
