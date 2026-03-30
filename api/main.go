package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	gw "github.com/hyperledger/fabric-samples/diaspora-api/internal/gateway"
	"github.com/hyperledger/fabric-samples/diaspora-api/internal/handlers"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("get home dir: %v", err)
	}

	orgsBase := filepath.Join(home, "fabric-samples", "test-network", "organizations", "peerOrganizations")

	orgConfigs := []gw.OrgConfig{
		{
			OrgName:      "Org1",
			MSPID:        "Org1MSP",
			CertPath:     filepath.Join(orgsBase, "org1.example.com", "users", "Admin@org1.example.com", "msp", "signcerts", "Admin@org1.example.com-cert.pem"),
			KeyPath:      filepath.Join(orgsBase, "org1.example.com", "users", "Admin@org1.example.com", "msp", "keystore"),
			TLSCertPath:  filepath.Join(orgsBase, "org1.example.com", "peers", "peer0.org1.example.com", "tls", "ca.crt"),
			PeerEndpoint: "dns:///localhost:7051",
			GatewayPeer:  "peer0.org1.example.com",
		},
		{
			OrgName:      "Org2",
			MSPID:        "Org2MSP",
			CertPath:     filepath.Join(orgsBase, "org2.example.com", "users", "Admin@org2.example.com", "msp", "signcerts", "Admin@org2.example.com-cert.pem"),
			KeyPath:      filepath.Join(orgsBase, "org2.example.com", "users", "Admin@org2.example.com", "msp", "keystore"),
			TLSCertPath:  filepath.Join(orgsBase, "org2.example.com", "peers", "peer0.org2.example.com", "tls", "ca.crt"),
			PeerEndpoint: "dns:///localhost:9051",
			GatewayPeer:  "peer0.org2.example.com",
		},
		{
			OrgName:      "Org3",
			MSPID:        "Org3MSP",
			CertPath:     filepath.Join(orgsBase, "org3.example.com", "users", "Admin@org3.example.com", "msp", "signcerts", "Admin@org3.example.com-cert.pem"),
			KeyPath:      filepath.Join(orgsBase, "org3.example.com", "users", "Admin@org3.example.com", "msp", "keystore"),
			TLSCertPath:  filepath.Join(orgsBase, "org3.example.com", "peers", "peer0.org3.example.com", "tls", "ca.crt"),
			PeerEndpoint: "dns:///localhost:11051",
			GatewayPeer:  "peer0.org3.example.com",
		},
	}

	orgConns := make(map[string]*gw.OrgConnection)
	for _, cfg := range orgConfigs {
		conn, err := gw.Initialize(cfg)
		if err != nil {
			log.Fatalf("initialize %s: %v", cfg.OrgName, err)
		}
		defer conn.Close()
		orgConns[cfg.OrgName] = conn
	}

	srv := handlers.NewServer(orgConns)
	mux := srv.Routes()

	addr := ":8080"
	fmt.Printf("Diaspora API listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
