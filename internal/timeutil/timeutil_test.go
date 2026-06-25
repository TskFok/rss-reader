package timeutil

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Setenv("TZ", "UTC")
	require.NoError(t, Init())
	require.NotNil(t, Location)
	assert.Equal(t, LocationName, Location.String())
	assert.Equal(t, LocationName, time.Local.String())
	assert.Equal(t, LocationName, os.Getenv("TZ"))
}

func TestNowUsesShanghaiAfterInit(t *testing.T) {
	require.NoError(t, Init())
	now := time.Now()
	_, offset := now.Zone()
	assert.Equal(t, 8*3600, offset)
}
