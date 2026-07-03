package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/certs"
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
)

var (
	certHosts []string
	certForce bool

	defaultCertHosts = []string{"localhost", "127.0.0.1"}
)

// getCertsDir resolves the certificate output directory under ~/.azd.
var getCertsDir = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".azd", "app", "certs"), nil
}

// NewCertCommand creates the cert command.
func NewCertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "Generate local HTTPS certificates",
		Long: `Generate local HTTPS certificates for development.

This command creates a local certificate authority and a TLS server certificate
under ~/.azd/app/certs. By default it includes localhost and 127.0.0.1.

Run it again to reuse existing valid certificates. Use --force to regenerate
the server certificate and key.

Examples:
  # Generate certs for localhost and 127.0.0.1
  azd app cert

  # Add extra hosts (repeat --host as needed)
  azd app cert --host api.local.test --host auth.local.test

  # Regenerate the server certificate
  azd app cert --force`,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE:         runCert,
	}

	cmd.Flags().StringSliceVar(&certHosts, "host", []string{}, "Additional host to include in certificate SANs (repeatable)")
	cmd.Flags().BoolVar(&certForce, "force", false, "Regenerate server certificate and key")

	return cmd
}

type certCommandOutput struct {
	CACertPath   string   `json:"caCertPath"`
	CertPath     string   `json:"certPath"`
	KeyPath      string   `json:"keyPath"`
	Hosts        []string `json:"hosts"`
	Reused       bool     `json:"reused"`
	TrustCommand string   `json:"trustCommand"`
}

func runCert(_ *cobra.Command, _ []string) error {
	hosts := append([]string{}, defaultCertHosts...)
	hosts = append(hosts, certHosts...)

	certDir, err := getCertsDir()
	if err != nil {
		return err
	}

	result, err := certs.Generate(certDir, hosts, certForce)
	if err != nil {
		return fmt.Errorf("generate certificates: %w", err)
	}

	trustCommand := trustCommandForOS(result.CACertPath)
	if cliout.IsJSON() {
		return cliout.PrintJSON(certCommandOutput{
			CACertPath:   result.CACertPath,
			CertPath:     result.CertPath,
			KeyPath:      result.KeyPath,
			Hosts:        result.Hosts,
			Reused:       result.Reused,
			TrustCommand: trustCommand,
		})
	}

	cliout.CommandHeader("cert", "Generate local HTTPS certificates")
	if result.Reused {
		cliout.Success("Reused existing local certificate files")
	} else {
		cliout.Success("Generated local certificate files")
	}
	cliout.Label("CA certificate", result.CACertPath)
	cliout.Label("TLS certificate", result.CertPath)
	cliout.Label("TLS private key", result.KeyPath)
	cliout.Label("Hosts", strings.Join(result.Hosts, ", "))
	cliout.Newline()
	cliout.Info("Trust the local CA:")
	cliout.Bullet("%s", trustCommand)

	return nil
}

func trustCommandForOS(caPath string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf(`certutil -addstore -user Root "%s"`, caPath)
	case "darwin":
		return fmt.Sprintf(`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "%s"`, caPath)
	case "linux":
		return fmt.Sprintf(`sudo cp "%s" /usr/local/share/ca-certificates/azd-app-local-ca.crt && sudo update-ca-certificates`, caPath)
	default:
		return fmt.Sprintf(`Import "%s" into your system trust store`, caPath)
	}
}
