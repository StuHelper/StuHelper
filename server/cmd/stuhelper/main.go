package main

import (
	"fmt"
	"os"

	"git.stuhelper.com/StuHelper/StuHelper/internal/app"
)

// 构建信息，通过 -ldflags 注入。
var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := app.Run(version, gitCommit, buildTime); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
