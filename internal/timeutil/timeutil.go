package timeutil

import (
	"fmt"
	"os"
	"time"
)

const (
	// LocationName 应用统一使用的 IANA 时区。
	LocationName = "Asia/Shanghai"
	// MySQLTimeZone MySQL 会话时区偏移（不依赖 mysql 时区表）。
	MySQLTimeZone = "+08:00"
)

// Location 上海时区；Init 成功后与 time.Local 一致。
var Location *time.Location

// Init 将进程时区固定为上海，供 time.Now() 与业务日期计算使用。
func Init() error {
	if err := os.Setenv("TZ", LocationName); err != nil {
		return fmt.Errorf("set TZ: %w", err)
	}
	loc, err := time.LoadLocation(LocationName)
	if err != nil {
		return fmt.Errorf("load location %s: %w", LocationName, err)
	}
	Location = loc
	time.Local = loc
	return nil
}
