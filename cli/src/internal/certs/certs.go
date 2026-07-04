// Package certs manages local certificate authority and TLS certificate files.
package certs

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	caCertFileName   = "ca.crt"
	caKeyFileName    = "ca.key"
	leafCertFileName = "cert.crt"
	leafKeyFileName  = "cert.key"

	caValidity   = 10 * 365 * 24 * time.Hour
	leafValidity = 825 * 24 * time.Hour
)

// Result contains generated certificate paths and metadata.
type Result struct {
	CACertPath string
	CertPath   string
	KeyPath    string
	Hosts      []string
	Reused     bool
}

// EnsureCA creates or loads a local certificate authority under dir.
func EnsureCA(dir string) (caCertPath, caKeyPath string, cert *x509.Certificate, key crypto.Signer, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", nil, nil, fmt.Errorf("create certificate directory: %w", err)
	}

	caCertPath = filepath.Join(dir, caCertFileName)
	caKeyPath = filepath.Join(dir, caKeyFileName)

	cert, key, err = loadCertificateAndKey(caCertPath, caKeyPath)
	if err == nil && cert.IsCA && cert.BasicConstraintsValid && isCertificateCurrentlyValid(cert) && keysMatch(cert.PublicKey, key.Public()) {
		return caCertPath, caKeyPath, cert, key, nil
	}

	cert, key, err = createCA(caCertPath, caKeyPath)
	if err != nil {
		return "", "", nil, nil, err
	}

	return caCertPath, caKeyPath, cert, key, nil
}

// IssueLeaf creates or reuses a TLS leaf certificate under dir for the given hosts.
func IssueLeaf(dir string, hosts []string, ca *x509.Certificate, caKey crypto.Signer, force bool) (Result, error) {
	if ca == nil {
		return Result{}, fmt.Errorf("ca certificate is required")
	}
	if caKey == nil {
		return Result{}, fmt.Errorf("ca private key is required")
	}

	normalizedHosts, err := normalizeHosts(hosts)
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Result{}, fmt.Errorf("create certificate directory: %w", err)
	}

	result := Result{
		CACertPath: filepath.Join(dir, caCertFileName),
		CertPath:   filepath.Join(dir, leafCertFileName),
		KeyPath:    filepath.Join(dir, leafKeyFileName),
		Hosts:      normalizedHosts,
	}

	if !force {
		cert, key, loadErr := loadCertificateAndKey(result.CertPath, result.KeyPath)
		if loadErr == nil && isLeafReusable(cert, key, normalizedHosts, ca) {
			result.Reused = true
			return result, nil
		}
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Result{}, fmt.Errorf("generate leaf private key: %w", err)
	}

	dnsNames, ipAddresses := splitHosts(normalizedHosts)
	serialNumber, err := randomSerialNumber()
	if err != nil {
		return Result{}, fmt.Errorf("generate leaf serial number: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   normalizedHosts[0],
			Organization: []string{"azd app local development"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(leafValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, template, ca, leafKey.Public(), caKey)
	if err != nil {
		return Result{}, fmt.Errorf("create leaf certificate: %w", err)
	}

	if err := writePEM(result.CertPath, "CERTIFICATE", leafDER, 0o644); err != nil {
		return Result{}, fmt.Errorf("write leaf certificate: %w", err)
	}
	if err := writePrivateKey(result.KeyPath, leafKey); err != nil {
		return Result{}, fmt.Errorf("write leaf private key: %w", err)
	}

	return result, nil
}

// Generate ensures a local CA exists and creates or reuses a leaf certificate.
func Generate(dir string, hosts []string, force bool) (Result, error) {
	caCertPath, _, caCert, caKey, err := EnsureCA(dir)
	if err != nil {
		return Result{}, err
	}

	result, err := IssueLeaf(dir, hosts, caCert, caKey, force)
	if err != nil {
		return Result{}, err
	}

	result.CACertPath = caCertPath
	return result, nil
}

func createCA(caCertPath, caKeyPath string) (*x509.Certificate, crypto.Signer, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ca private key: %w", err)
	}

	serialNumber, err := randomSerialNumber()
	if err != nil {
		return nil, nil, fmt.Errorf("generate ca serial number: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "azd app local development CA",
			Organization: []string{"azd app local development"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(caValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, template, template, caKey.Public(), caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create ca certificate: %w", err)
	}

	if err := writePEM(caCertPath, "CERTIFICATE", caDER, 0o644); err != nil {
		return nil, nil, fmt.Errorf("write ca certificate: %w", err)
	}
	if err := writePrivateKey(caKeyPath, caKey); err != nil {
		return nil, nil, fmt.Errorf("write ca private key: %w", err)
	}

	cert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parse generated ca certificate: %w", err)
	}

	return cert, caKey, nil
}

func normalizeHosts(hosts []string) ([]string, error) {
	normalized := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))

	for _, host := range hosts {
		cleanHost := strings.TrimSpace(host)
		if cleanHost == "" {
			continue
		}

		if strings.HasPrefix(cleanHost, "[") && strings.HasSuffix(cleanHost, "]") {
			cleanHost = strings.TrimPrefix(strings.TrimSuffix(cleanHost, "]"), "[")
		}

		if ip := net.ParseIP(cleanHost); ip != nil {
			cleanHost = ip.String()
		} else {
			cleanHost = strings.ToLower(cleanHost)
			if strings.Contains(cleanHost, "/") {
				return nil, fmt.Errorf("invalid host %q: path separators are not allowed", host)
			}
			if strings.Contains(cleanHost, ":") {
				return nil, fmt.Errorf("invalid host %q: include host names only, no ports", host)
			}
		}

		if _, exists := seen[cleanHost]; exists {
			continue
		}

		seen[cleanHost] = struct{}{}
		normalized = append(normalized, cleanHost)
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("at least one host is required")
	}

	sort.Strings(normalized)
	return normalized, nil
}

func splitHosts(hosts []string) ([]string, []net.IP) {
	dnsNames := make([]string, 0, len(hosts))
	ipAddresses := make([]net.IP, 0, len(hosts))

	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			ipAddresses = append(ipAddresses, ip)
			continue
		}
		dnsNames = append(dnsNames, host)
	}

	return dnsNames, ipAddresses
}

func isLeafReusable(cert *x509.Certificate, key crypto.Signer, hosts []string, ca *x509.Certificate) bool {
	if cert == nil || key == nil {
		return false
	}

	if !isCertificateCurrentlyValid(cert) {
		return false
	}

	if err := cert.CheckSignatureFrom(ca); err != nil {
		return false
	}

	if !hostsCovered(cert, hosts) {
		return false
	}

	if !hasServerAuth(cert.ExtKeyUsage) {
		return false
	}

	return keysMatch(cert.PublicKey, key.Public())
}

func isCertificateCurrentlyValid(cert *x509.Certificate) bool {
	now := time.Now()
	return !now.Before(cert.NotBefore) && !now.After(cert.NotAfter)
}

func hostsCovered(cert *x509.Certificate, hosts []string) bool {
	for _, host := range hosts {
		if !hostCovered(cert, host) {
			return false
		}
	}
	return true
}

func hostCovered(cert *x509.Certificate, host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		for _, certIP := range cert.IPAddresses {
			if certIP.Equal(ip) {
				return true
			}
		}
		return false
	}

	for _, dnsName := range cert.DNSNames {
		if strings.EqualFold(dnsName, host) {
			return true
		}
	}

	return false
}

func hasServerAuth(extKeyUsages []x509.ExtKeyUsage) bool {
	for _, usage := range extKeyUsages {
		if usage == x509.ExtKeyUsageServerAuth {
			return true
		}
	}
	return false
}

func keysMatch(certPublicKey any, privatePublicKey any) bool {
	certDER, certErr := x509.MarshalPKIXPublicKey(certPublicKey)
	if certErr != nil {
		return false
	}

	keyDER, keyErr := x509.MarshalPKIXPublicKey(privatePublicKey)
	if keyErr != nil {
		return false
	}

	return bytes.Equal(certDER, keyDER)
}

func randomSerialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, err
	}
	if serialNumber.Sign() <= 0 {
		return nil, fmt.Errorf("generated non-positive serial number")
	}
	return serialNumber, nil
}

func loadCertificateAndKey(certPath, keyPath string) (*x509.Certificate, crypto.Signer, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read certificate %q: %w", certPath, err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read private key %q: %w", keyPath, err)
	}

	cert, err := parseCertificatePEM(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse certificate %q: %w", certPath, err)
	}

	key, err := parsePrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key %q: %w", keyPath, err)
	}

	return cert, key, nil
}

func parseCertificatePEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("missing PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected PEM type %q", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func parsePrivateKeyPEM(keyPEM []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("missing PEM block")
	}

	switch block.Type {
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		signer, ok := parsed.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("private key is not a crypto.Signer")
		}

		return signer, nil
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported private key PEM type %q", block.Type)
	}
}

func writePrivateKey(path string, key crypto.Signer) error {
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}

	return writePEM(path, "PRIVATE KEY", keyBytes, 0o600)
}

func writePEM(path, pemType string, derBytes []byte, mode os.FileMode) error {
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  pemType,
		Bytes: derBytes,
	})

	if err := os.WriteFile(path, pemBytes, mode); err != nil {
		return err
	}

	if err := os.Chmod(path, mode); err != nil {
		return err
	}

	return nil
}
