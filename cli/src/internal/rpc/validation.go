package rpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jongio/azd-app/cli/src/internal/azure"
)

const (
	// maxQueryBytes caps the size of any KQL query persisted to azure.yaml.
	// 8 KiB is large enough for any real query while bounding the blast radius
	// of attacker-supplied content (CWE-915/94).
	maxQueryBytes = 8 * 1024
)

// knownAzureTableSet is the server-side allowlist for table names that may be
// written into azure.yaml. Populated once at package init from
// azure.TableCategories so the canonical source of truth lives in the azure
// package and this file stays thin.
var knownAzureTableSet map[string]struct{}

func init() {
	knownAzureTableSet = make(map[string]struct{})
	for _, cat := range azure.TableCategories {
		for _, name := range cat.Tables {
			knownAzureTableSet[name] = struct{}{}
		}
	}
}

// validateQuery rejects KQL query strings that exceed the size limit or
// contain non-printable bytes (other than the standard whitespace control
// codes \n, \r, \t). Call sites translate the returned error directly to
// connect.CodeInvalidArgument.
//
// Two invariants it enforces:
//  1. Size ≤ maxQueryBytes — prevents gigantic payloads from bloating or
//     corrupting azure.yaml.
//  2. No non-printable bytes (except \n, \r, \t) — prevents null-byte
//     injection and other control-code tricks that could confuse YAML
//     parsers or downstream KQL evaluation (CWE-94).
func validateQuery(query string) error {
	if len(query) > maxQueryBytes {
		return fmt.Errorf("query exceeds %d-byte limit (got %d bytes)", maxQueryBytes, len(query))
	}
	for i, b := range []byte(query) {
		// Allow standard whitespace control codes. Reject everything else
		// below 0x20 (control range) and 0x7f (DEL).
		if (b < 0x20 && b != '\n' && b != '\r' && b != '\t') || b == 0x7f {
			return fmt.Errorf("query contains non-printable byte 0x%02x at offset %d", b, i)
		}
	}
	return nil
}

// validateTables rejects table names that are absent from the server-side
// allowlist of known Azure Log Analytics tables. Unknown names are blocked to
// prevent injection of arbitrary content into azure.yaml (CWE-915).
func validateTables(tables []string) error {
	for _, name := range tables {
		if _, ok := knownAzureTableSet[name]; !ok {
			return fmt.Errorf("unknown table %q: not in Azure Log Analytics allowlist", name)
		}
	}
	return nil
}

// auditMutation emits a structured slog.Info audit entry for every RPC that
// persists caller-supplied content to azure.yaml. op identifies the handler;
// peer is the caller's network address from connect.Request.Peer().Addr (empty
// in in-process tests and when the transport does not surface peer info).
// extra is an optional flat key-value slice appended verbatim to the record.
func auditMutation(ctx context.Context, op, peer string, extra ...any) {
	args := make([]any, 0, 4+len(extra))
	args = append(args, "op", op, "peer", peer)
	args = append(args, extra...)
	slog.InfoContext(ctx, "rpc mutation audit", args...)
}
