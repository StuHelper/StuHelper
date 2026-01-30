// Package migrate 提供数据库迁移功能
package migrate

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

// Config 迁移配置
type Config struct {
	DatabaseURL string
	Dir         string // 可选，默认使用嵌入的迁移文件
}

// Run 执行数据库迁移
func Run(db *sql.DB, command string) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	switch command {
	case "up":
		return goose.Up(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	case "version":
		return goose.Version(db, ".")
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// Up 执行所有待执行的迁移
func Up(db *sql.DB) error {
	return Run(db, "up")
}

// Down 回滚最近一次迁移
func Down(db *sql.DB) error {
	return Run(db, "down")
}

// Status 显示迁移状态
func Status(db *sql.DB) error {
	return Run(db, "status")
}
