package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureMySQLDSNTimezone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "append loc",
			in:   "user:pass@tcp(localhost:3306)/rss_reader?charset=utf8mb4&parseTime=True",
			want: "user:pass@tcp(localhost:3306)/rss_reader?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai",
		},
		{
			name: "keep existing loc",
			in:   "user:pass@tcp(localhost:3306)/db?loc=UTC",
			want: "user:pass@tcp(localhost:3306)/db?loc=UTC",
		},
		{
			name: "non mysql unchanged",
			in:   "file:test.db?mode=memory",
			want: "file:test.db?mode=memory",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ensureMySQLDSNTimezone(tc.in))
		})
	}
}

func TestIsMySQLDSN(t *testing.T) {
	t.Parallel()
	assert.True(t, isMySQLDSN("user:pass@tcp(localhost:3306)/db"))
	assert.True(t, isMySQLDSN("mysql:user:pass@tcp(localhost:3306)/db"))
	assert.False(t, isMySQLDSN("file:test.db"))
}
