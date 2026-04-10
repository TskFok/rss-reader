package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestArticleAutoMigrate_HasAIColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:schema_check?mode=memory"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Article{}))
	var cols []string
	rows, err := db.Raw("SELECT name FROM pragma_table_info('articles')").Rows()
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		cols = append(cols, n)
	}
	assert.Contains(t, cols, "ai_process_status")
	assert.NotContains(t, cols, "a_iprocess_status")
	assert.Contains(t, cols, "ai_category")
	assert.Contains(t, cols, "guid_raw")
}
