package db

import "testing"

func TestDatabaseConnection(t *testing.T) {
	sqlDB, err := GetDB().DB()
	if err != nil {
		t.Fatalf("获取数据库连接失败：%v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("MySQL Ping 失败：%v", err)
	}
}
