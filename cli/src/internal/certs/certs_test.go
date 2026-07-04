package certs

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCA_GeneratesParseableCA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	caCertPath, caKeyPath, caCert, caKey, err := EnsureCA(dir)
	require.NoError(t, err)

	require.FileExists(t, caCertPath)
	require.FileExists(t, caKeyPath)
	require.NotNil(t, caCert)
	require.NotNil(t, caKey)
	assert.True(t, caCert.IsCA)
	assert.True(t, caCert.BasicConstraintsValid)

	diskCert := readCertificateFromFile(t, caCertPath)
	assert.True(t, diskCert.IsCA)
	assert.True(t, diskCert.BasicConstraintsValid)

	_, err = readSignerFromFile(caKeyPath)
	require.NoError(t, err)
}

func TestIssueLeaf_IncludesHostsAndVerifiesAgainstCA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, _, caCert, caKey, err := EnsureCA(dir)
	require.NoError(t, err)

	hosts := []string{"localhost", "127.0.0.1", "api.local.test"}
	result, err := IssueLeaf(dir, hosts, caCert, caKey, false)
	require.NoError(t, err)

	assert.False(t, result.Reused)
	assert.Equal(t, []string{"127.0.0.1", "api.local.test", "localhost"}, result.Hosts)
	require.FileExists(t, result.CertPath)
	require.FileExists(t, result.KeyPath)

	leafCert := readCertificateFromFile(t, result.CertPath)
	assert.ElementsMatch(t, []string{"api.local.test", "localhost"}, leafCert.DNSNames)
	assert.True(t, containsIP(leafCert.IPAddresses, "127.0.0.1"))
	assert.Contains(t, leafCert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

	rootPool := x509.NewCertPool()
	rootPool.AddCert(caCert)

	_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootPool})
	require.NoError(t, err)
}

func TestIssueLeaf_HostCoverageSplitsDNSAndIP(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, _, caCert, caKey, err := EnsureCA(dir)
	require.NoError(t, err)

	hosts := []string{"LOCALHOST", "127.0.0.1", "::1", "api.internal.local"}
	result, err := IssueLeaf(dir, hosts, caCert, caKey, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"127.0.0.1", "::1", "api.internal.local", "localhost"}, result.Hosts)

	leafCert := readCertificateFromFile(t, result.CertPath)
	assert.ElementsMatch(t, []string{"api.internal.local", "localhost"}, leafCert.DNSNames)

	for _, ip := range []string{"127.0.0.1", "::1"} {
		assert.Truef(t, containsIP(leafCert.IPAddresses, ip), "expected ip %s in SANs", ip)
	}
}

func TestGenerate_ReusesCertificatesAndForceRegeneratesLeaf(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hosts := []string{"localhost", "127.0.0.1"}

	first, err := Generate(dir, hosts, false)
	require.NoError(t, err)
	assert.False(t, first.Reused)

	firstCABytes, err := os.ReadFile(first.CACertPath)
	require.NoError(t, err)
	firstLeaf := readCertificateFromFile(t, first.CertPath)

	second, err := Generate(dir, hosts, false)
	require.NoError(t, err)
	assert.True(t, second.Reused)

	secondCABytes, err := os.ReadFile(second.CACertPath)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(firstCABytes, secondCABytes))

	secondLeaf := readCertificateFromFile(t, second.CertPath)
	assert.Zero(t, firstLeaf.SerialNumber.Cmp(secondLeaf.SerialNumber))

	third, err := Generate(dir, []string{"localhost", "127.0.0.1", "api.local.test"}, false)
	require.NoError(t, err)
	assert.False(t, third.Reused)

	thirdLeaf := readCertificateFromFile(t, third.CertPath)
	assert.NotZero(t, secondLeaf.SerialNumber.Cmp(thirdLeaf.SerialNumber))
	assert.Contains(t, thirdLeaf.DNSNames, "api.local.test")

	forced, err := Generate(dir, hosts, true)
	require.NoError(t, err)
	assert.False(t, forced.Reused)

	forcedLeaf := readCertificateFromFile(t, forced.CertPath)
	assert.NotZero(t, thirdLeaf.SerialNumber.Cmp(forcedLeaf.SerialNumber))

	forcedCABytes, err := os.ReadFile(forced.CACertPath)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(secondCABytes, forcedCABytes))
}

func readCertificateFromFile(t *testing.T, path string) *x509.Certificate {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	block, _ := pem.Decode(data)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}

func readSignerFromFile(path string) (crypto.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parsePrivateKeyPEM(data)
}

func containsIP(ips []net.IP, expected string) bool {
	target := net.ParseIP(expected)
	if target == nil {
		return false
	}

	for _, ip := range ips {
		if ip.Equal(target) {
			return true
		}
	}
	return false
}

func TestEnsureCA_ReusesExistingCA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	caCertPath, _, firstCert, _, err := EnsureCA(dir)
	require.NoError(t, err)
	firstBytes, err := os.ReadFile(caCertPath)
	require.NoError(t, err)

	_, _, secondCert, secondKey, err := EnsureCA(dir)
	require.NoError(t, err)
	require.NotNil(t, secondCert)
	require.NotNil(t, secondKey)

	secondBytes, err := os.ReadFile(caCertPath)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(firstBytes, secondBytes), "CA file should not be rewritten when reused")
	assert.Zero(t, firstCert.SerialNumber.Cmp(secondCert.SerialNumber), "reused CA should keep the same serial")
	assert.True(t, keysMatch(secondCert.PublicKey, secondKey.Public()))
}

func TestEnsureCA_WriteError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Make the CA cert path a directory so writing (and reading) the CA file fails.
	require.NoError(t, os.Mkdir(filepath.Join(dir, caCertFileName), 0o755))

	_, _, _, _, err := EnsureCA(dir)
	require.Error(t, err)
}

func TestIssueLeaf_Errors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, _, caCert, caKey, err := EnsureCA(dir)
	require.NoError(t, err)

	t.Run("nil ca", func(t *testing.T) {
		t.Parallel()
		_, err := IssueLeaf(dir, []string{"localhost"}, nil, caKey, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ca certificate is required")
	})

	t.Run("nil ca key", func(t *testing.T) {
		t.Parallel()
		_, err := IssueLeaf(dir, []string{"localhost"}, caCert, nil, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ca private key is required")
	})

	t.Run("no hosts", func(t *testing.T) {
		t.Parallel()
		_, err := IssueLeaf(dir, []string{"   ", ""}, caCert, caKey, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one host is required")
	})
}

func TestGenerate_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("ensure ca fails when dir is a file", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()
		fileAsDir := filepath.Join(base, "not-a-dir")
		require.NoError(t, os.WriteFile(fileAsDir, []byte("x"), 0o600))

		_, err := Generate(fileAsDir, []string{"localhost"}, false)
		require.Error(t, err)
	})

	t.Run("issue leaf fails with invalid host", func(t *testing.T) {
		t.Parallel()
		_, err := Generate(t.TempDir(), []string{"host:8080"}, false)
		require.Error(t, err)
	})
}

func TestNormalizeHosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      []string
		want    []string
		wantErr string
	}{
		{
			name: "dedup lowercase sort and trim",
			in:   []string{"B.test", "a.test", "A.TEST", " a.test ", "  "},
			want: []string{"a.test", "b.test"},
		},
		{
			name: "bracketed ipv6 is unwrapped",
			in:   []string{"[::1]"},
			want: []string{"::1"},
		},
		{
			name: "ipv4 preserved",
			in:   []string{"127.0.0.1"},
			want: []string{"127.0.0.1"},
		},
		{
			name:    "path separator rejected",
			in:      []string{"foo/bar"},
			wantErr: "path separators are not allowed",
		},
		{
			name:    "port rejected",
			in:      []string{"host:8080"},
			wantErr: "no ports",
		},
		{
			name:    "all empty rejected",
			in:      []string{"", "   "},
			wantErr: "at least one host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeHosts(tt.in)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadCertificateAndKey_Errors(t *testing.T) {
	t.Parallel()

	// Produce a known-good cert/key pair to mix with invalid inputs.
	srcDir := t.TempDir()
	goodCertPath, goodKeyPath, _, _, err := EnsureCA(srcDir)
	require.NoError(t, err)
	goodCert, err := os.ReadFile(goodCertPath)
	require.NoError(t, err)
	goodKey, err := os.ReadFile(goodKeyPath)
	require.NoError(t, err)

	t.Run("missing certificate file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, _, err := loadCertificateAndKey(filepath.Join(dir, "cert.crt"), filepath.Join(dir, "cert.key"))
		require.Error(t, err)
	})

	t.Run("missing key file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.crt")
		require.NoError(t, os.WriteFile(certPath, goodCert, 0o600))
		_, _, err := loadCertificateAndKey(certPath, filepath.Join(dir, "cert.key"))
		require.Error(t, err)
	})

	t.Run("invalid certificate contents", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.crt")
		keyPath := filepath.Join(dir, "cert.key")
		require.NoError(t, os.WriteFile(certPath, []byte("not a cert"), 0o600))
		require.NoError(t, os.WriteFile(keyPath, goodKey, 0o600))
		_, _, err := loadCertificateAndKey(certPath, keyPath)
		require.Error(t, err)
	})

	t.Run("invalid key contents", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.crt")
		keyPath := filepath.Join(dir, "cert.key")
		require.NoError(t, os.WriteFile(certPath, goodCert, 0o600))
		require.NoError(t, os.WriteFile(keyPath, []byte("not a key"), 0o600))
		_, _, err := loadCertificateAndKey(certPath, keyPath)
		require.Error(t, err)
	})
}

func TestParseCertificatePEM_Errors(t *testing.T) {
	t.Parallel()

	t.Run("missing pem block", func(t *testing.T) {
		t.Parallel()
		_, err := parseCertificatePEM([]byte("not pem data"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing PEM block")
	})

	t.Run("unexpected pem type", func(t *testing.T) {
		t.Parallel()
		block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{0x01, 0x02}})
		_, err := parseCertificatePEM(block)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected PEM type")
	})

	t.Run("undecodable certificate bytes", func(t *testing.T) {
		t.Parallel()
		block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte{0x01, 0x02, 0x03}})
		_, err := parseCertificatePEM(block)
		require.Error(t, err)
	})
}

func TestParsePrivateKeyPEM(t *testing.T) {
	t.Parallel()

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pkcs8DER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rsaDER := x509.MarshalPKCS1PrivateKey(rsaKey)

	block := func(pemType string, der []byte) []byte {
		return pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: der})
	}

	tests := []struct {
		name    string
		pem     []byte
		wantErr bool
	}{
		{name: "pkcs8", pem: block("PRIVATE KEY", pkcs8DER)},
		{name: "ec", pem: block("EC PRIVATE KEY", ecDER)},
		{name: "rsa", pem: block("RSA PRIVATE KEY", rsaDER)},
		{name: "missing block", pem: []byte("nope"), wantErr: true},
		{name: "unsupported type", pem: block("DSA PRIVATE KEY", []byte{0x01}), wantErr: true},
		{name: "pkcs8 garbage", pem: block("PRIVATE KEY", []byte{0x01, 0x02}), wantErr: true},
		{name: "ec garbage", pem: block("EC PRIVATE KEY", []byte{0x01, 0x02}), wantErr: true},
		{name: "rsa garbage", pem: block("RSA PRIVATE KEY", []byte{0x01, 0x02}), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			signer, err := parsePrivateKeyPEM(tt.pem)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, signer)
		})
	}
}

func TestIsLeafReusable_RejectsInvalidCandidates(t *testing.T) {
	t.Parallel()

	ca, caKey := newTestCA(t)
	otherCA, _ := newTestCA(t)
	hosts := []string{"127.0.0.1", "localhost"}
	now := time.Now()

	validCert, validKey := signTestLeaf(t, ca, caKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, hosts)
	require.True(t, isLeafReusable(validCert, validKey, hosts, ca), "freshly signed cert should be reusable")

	assert.False(t, isLeafReusable(nil, validKey, hosts, ca), "nil cert")
	assert.False(t, isLeafReusable(validCert, nil, hosts, ca), "nil key")

	expiredCert, expiredKey := signTestLeaf(t, ca, caKey, now.Add(-2*time.Hour), now.Add(-time.Hour), []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, hosts)
	assert.False(t, isLeafReusable(expiredCert, expiredKey, hosts, ca), "expired cert")

	assert.False(t, isLeafReusable(validCert, validKey, hosts, otherCA), "signed by a different CA")

	assert.False(t, isLeafReusable(validCert, validKey, []string{"missing.host"}, ca), "host not covered")

	noAuthCert, noAuthKey := signTestLeaf(t, ca, caKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, hosts)
	assert.False(t, isLeafReusable(noAuthCert, noAuthKey, hosts, ca), "missing server auth usage")

	_, otherKey := signTestLeaf(t, ca, caKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, hosts)
	assert.False(t, isLeafReusable(validCert, otherKey, hosts, ca), "key does not match cert")
}

func TestHasServerAuth(t *testing.T) {
	t.Parallel()

	assert.True(t, hasServerAuth([]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}))
	assert.False(t, hasServerAuth([]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}))
	assert.False(t, hasServerAuth(nil))
}

func TestHostCovered(t *testing.T) {
	t.Parallel()

	ca, caKey := newTestCA(t)
	now := time.Now()
	cert, _ := signTestLeaf(t, ca, caKey, now.Add(-time.Hour), now.Add(time.Hour), []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, []string{"example.test", "127.0.0.1"})

	assert.True(t, hostCovered(cert, "example.test"))
	assert.True(t, hostCovered(cert, "EXAMPLE.TEST"), "DNS match should be case-insensitive")
	assert.True(t, hostCovered(cert, "127.0.0.1"))
	assert.False(t, hostCovered(cert, "missing.test"), "unknown DNS name")
	assert.False(t, hostCovered(cert, "10.0.0.1"), "unknown IP address")
}

func TestKeysMatch(t *testing.T) {
	t.Parallel()

	k1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	k2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	assert.True(t, keysMatch(k1.Public(), k1.Public()))
	assert.False(t, keysMatch(k1.Public(), k2.Public()))
	// Unmarshalable public keys must be treated as non-matching, not panic.
	assert.False(t, keysMatch("not-a-key", k1.Public()))
	assert.False(t, keysMatch(k1.Public(), "not-a-key"))
}

func TestWritePEM_WriteError(t *testing.T) {
	t.Parallel()

	// Writing to an existing directory path fails on every OS.
	dir := t.TempDir()
	err := writePEM(dir, "CERTIFICATE", []byte{0x01, 0x02}, 0o644)
	require.Error(t, err)
}

func newTestCA(t *testing.T) (*x509.Certificate, crypto.Signer) {
	t.Helper()

	_, _, cert, key, err := EnsureCA(t.TempDir())
	require.NoError(t, err)
	return cert, key
}

func signTestLeaf(t *testing.T, ca *x509.Certificate, caKey crypto.Signer, notBefore, notAfter time.Time, usages []x509.ExtKeyUsage, hosts []string) (*x509.Certificate, crypto.Signer) {
	t.Helper()

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	var dnsNames []string
	var ipAddresses []net.IP
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			ipAddresses = append(ipAddresses, ip)
			continue
		}
		dnsNames = append(dnsNames, host)
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "test leaf"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           usages,
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca, leafKey.Public(), caKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, leafKey
}
