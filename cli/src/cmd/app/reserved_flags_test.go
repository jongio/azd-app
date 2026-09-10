package main

import (
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoReservedFlagConflicts guards against an extension flag colliding with an
// azd reserved global flag. azdext.Run performs the same check at startup, so
// without this test the failure would only surface on a user's first run. This
// extension registers the largest command tree of the three, so it is the most
// likely to trip the check.
func TestNoReservedFlagConflicts(t *testing.T) {
	require.NoError(t, azdext.ValidateNoReservedFlagConflicts(newRootCmd()))
}

// TestReservedFlagNamesIsNotEmpty keeps the check above from silently becoming
// vacuous if the SDK ever ships an empty reserved-flag set.
func TestReservedFlagNamesIsNotEmpty(t *testing.T) {
	assert.NotEmpty(t, azdext.ReservedFlagNames())
}
