package database

import "strings"

const mysqlDSNLocParam = "loc=Asia%2FShanghai"

// ensureMySQLDSNTimezone 为 MySQL DSN 补齐 loc，使 parseTime 与业务时区一致。
func ensureMySQLDSNTimezone(dsn string) string {
	if !isMySQLDSN(dsn) {
		return dsn
	}
	if strings.Contains(dsn, "loc=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + mysqlDSNLocParam
}

func isMySQLDSN(dsn string) bool {
	return strings.Contains(dsn, "tcp(") || strings.HasPrefix(dsn, "mysql:")
}
