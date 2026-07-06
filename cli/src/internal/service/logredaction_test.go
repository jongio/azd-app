package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterLogRedactionCustomPattern(t *testing.T) {
	t.Cleanup(ResetLogRedaction)

	errs := RegisterLogRedaction([]string{`acme-[a-z0-9]{6}`}, nil)
	require.Empty(t, errs)

	got := MaskSecretsInLogLine("connecting with token acme-ab12cd for user bob")
	assert.NotContains(t, got, "acme-ab12cd")
	assert.Contains(t, got, "***")
	assert.Contains(t, got, "for user bob")
}

func TestRegisterLogRedactionLiteral(t *testing.T) {
	t.Cleanup(ResetLogRedaction)

	errs := RegisterLogRedaction(nil, []string{"super-secret-value"})
	require.Empty(t, errs)

	got := MaskSecretsInLogLine("value is super-secret-value here")
	assert.NotContains(t, got, "super-secret-value")
	assert.Contains(t, got, "***")
}

func TestRegisterLogRedactionInvalidPatternSkipped(t *testing.T) {
	t.Cleanup(ResetLogRedaction)

	errs := RegisterLogRedaction([]string{`valid-[0-9]+`, `(unclosed`}, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "invalid redaction pattern")

	// The valid pattern is still installed despite the invalid one.
	got := MaskSecretsInLogLine("id valid-42 done")
	assert.NotContains(t, got, "valid-42")
	assert.Contains(t, got, "***")
}

func TestMaskSecretsInLogLineBuiltInStillAppliesWithNoConfig(t *testing.T) {
	ResetLogRedaction()

	got := MaskSecretsInLogLine("password=hunter2secret rest")
	assert.Contains(t, got, "password=***")
	assert.NotContains(t, got, "hunter2secret")
}

func TestRegisterLogRedactionEmptyEntriesIgnored(t *testing.T) {
	t.Cleanup(ResetLogRedaction)

	errs := RegisterLogRedaction([]string{"", "   "}, []string{""})
	assert.Empty(t, errs)

	got := MaskSecretsInLogLine("nothing to mask here")
	assert.Equal(t, "nothing to mask here", got)
}
