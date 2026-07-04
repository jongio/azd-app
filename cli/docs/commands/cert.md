# azd app cert

Generate local HTTPS certificates.

## Synopsis

```
azd app cert [flags]
```

## Description

Generate a local certificate authority (CA) and a TLS server certificate for local development.

Certificates are stored in `~/.azd/app/certs`:

- `ca.crt` - Local CA certificate
- `ca.key` - Local CA private key
- `cert.crt` - TLS server certificate
- `cert.key` - TLS server private key

By default, the server certificate includes `localhost` and `127.0.0.1`.
Use `--host` to add more SAN entries.

When valid files already exist, the command reuses them.
Use `--force` to regenerate the server certificate and key.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--host` | | stringSlice | `[]` | Additional host to include in certificate SANs (repeatable) |
| `--force` | | bool | `false` | Regenerate server certificate and key |

## Examples

### Generate default local certificates

```bash
azd app cert
```

### Add extra hosts

```bash
azd app cert --host api.local.test --host auth.local.test
```

### Regenerate the server certificate

```bash
azd app cert --force
```

### Use generated paths in a service

```bash
# Linux/macOS
export HTTPS_CERT_FILE="$HOME/.azd/app/certs/cert.crt"
export HTTPS_KEY_FILE="$HOME/.azd/app/certs/cert.key"
```

```powershell
# Windows PowerShell
$env:HTTPS_CERT_FILE = "$HOME\.azd\app\certs\cert.crt"
$env:HTTPS_KEY_FILE = "$HOME\.azd\app\certs\cert.key"
```

## Trust the local CA

The command prints the trust command for your OS. You can also run one of these directly:

```powershell
# Windows
certutil -addstore -user Root "$HOME\.azd\app\certs\ca.crt"
```

```bash
# macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$HOME/.azd/app/certs/ca.crt"
```

```bash
# Linux
sudo cp "$HOME/.azd/app/certs/ca.crt" /usr/local/share/ca-certificates/azd-app-local-ca.crt
sudo update-ca-certificates
```

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | Certificates were generated or reused successfully |
| `1` | Certificate generation failed |

## Related Commands

- [azd app run](run.md) - Run the development environment
- [azd app start](start.md) - Start stopped services
- [azd app restart](restart.md) - Restart services
