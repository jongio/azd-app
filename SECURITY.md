# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of azd-app seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please do NOT:

- Open a public GitHub issue
- Disclose the vulnerability publicly before it has been addressed

### Please DO:


**How to report a vulnerability:**

1. File a private issue in this repository and mark it as security-related.
2. Include:
   - Type of vulnerability
   - Full paths of source file(s) related to the manifestation of the vulnerability
   - Location of the affected source code (tag/branch/commit or direct URL)
   - Step-by-step instructions to reproduce the issue
   - Proof-of-concept or exploit code (if possible)
   - Impact of the vulnerability, including how an attacker might exploit it

### What to Expect:

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours
- **Communication**: We will send you regular updates about our progress
- **Timeline**: We aim to patch critical vulnerabilities within 7 days
- **Credit**: If you would like, we will credit you in our release notes

### Security Best Practices

When using azd-app:

1. **Keep Updated**: Always use the latest version
2. **Validate Inputs**: Never run commands with untrusted input
3. **Review Permissions**: Ensure proper file and directory permissions
4. **Environment Variables**: Protect sensitive environment variables
5. **Azure Credentials**: Never commit credentials to source control

### Security Features

azd-app implements several security measures:

- **Input Validation**: All file paths are validated to prevent path traversal
- **Command Sanitization**: Script names are sanitized to prevent injection
- **Secure Random**: Cryptographically secure random number generation
- **Timeout Protection**: Commands have timeouts to prevent hung processes
- **Error Handling**: Comprehensive error handling with proper cleanup

For more details, see [docs/dev/security-status.md](cli/docs/dev/security-status.md).

## Security Scanning

We use the following tools to maintain security:

- **gosec**: Go security checker
- **Go vulnerability database**: Regular dependency scanning
- **GitHub Dependabot**: Automated dependency updates

## Learn More

- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [OWASP Go Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Go_Security_Cheat_Sheet.html)
