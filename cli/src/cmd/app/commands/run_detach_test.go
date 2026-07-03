package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripDetachFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "removes bare detach flag",
			args: []string{"run", "--detach", "--runtime", "azd"},
			want: []string{"run", "--runtime", "azd"},
		},
		{
			name: "removes key value detach flag",
			args: []string{"run", "--detach=true", "--service", "api"},
			want: []string{"run", "--service", "api"},
		},
		{
			name: "leaves args when no detach flag",
			args: []string{"run", "--runtime", "aspire"},
			want: []string{"run", "--runtime", "aspire"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripDetachFlag(tt.args))
		})
	}
}
