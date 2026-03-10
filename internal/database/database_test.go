package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// 验证无效 DSN 时返回错误
	_, err := Init("invalid-dsn")
	require.Error(t, err)
	assert.NotNil(t, err)
}
