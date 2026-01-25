package main

import (
	"fmt"
	"os"

	"github.com/zhuangbiaowei/LocalAIStack/internal/cli"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("Error: %v", err))
		os.Exit(1)
	}
}
