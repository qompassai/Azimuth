package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qompassai/azimuth/internal/config"
)

// writeCertPair generates a self-signed certificate and writes it
// (plus a CA bundle containing it) into dir, returning a Config
// that points at the three files LoadTLS needs.
func writeCertPair(t *testing.T, dir string) *config.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "azimuth-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.WriteFile(caPath, certPEM, 0o600); err != nil {
		t.Fatalf("write ca: %v", err)
	}
	return &config.Config{TLSCert: certPath, TLSKey: keyPath, TLSCA: caPath}
}

// Validation: a real cert pair and CA bundle load into a TLS 1.3
// configuration with verification on.
func TestLoadTLSValid(t *testing.T) {
	cfg := writeCertPair(t, t.TempDir())
	tlsCfg, err := LoadTLS(cfg)
	if err != nil {
		t.Fatalf("LoadTLS: %v", err)
	}
	if tlsCfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %x, want TLS 1.3 (%x)", tlsCfg.MinVersion, tls.VersionTLS13)
	}
	if tlsCfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify is true, want false")
	}
	if len(tlsCfg.Certificates) != 1 || tlsCfg.RootCAs == nil {
		t.Errorf("incomplete config: %+v", tlsCfg)
	}
}

// Adversarial: nil config, missing paths, missing files, and a CA
// bundle with no valid certificates are all errors, never a
// half-built configuration.
func TestLoadTLSRejects(t *testing.T) {
	dir := t.TempDir()
	good := writeCertPair(t, dir)
	badCA := &config.Config{
		TLSCert: good.TLSCert, TLSKey: good.TLSKey,
		TLSCA: filepath.Join(dir, "bad-ca.pem"),
	}
	if err := os.WriteFile(badCA.TLSCA, []byte("not a pem bundle"), 0o600); err != nil {
		t.Fatalf("write bad ca: %v", err)
	}
	cases := map[string]*config.Config{
		"nil config":    nil,
		"empty paths":   {},
		"missing files": {TLSCert: "/nonexistent/c", TLSKey: "/nonexistent/k", TLSCA: "/nonexistent/a"},
		"bad ca bundle": badCA,
	}
	for name, cfg := range cases {
		if _, err := LoadTLS(cfg); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}
