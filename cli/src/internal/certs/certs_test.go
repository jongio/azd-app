package certs

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"testing"

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
