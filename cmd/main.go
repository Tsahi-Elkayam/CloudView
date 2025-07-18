package main

import (
	"os"

	"github.com/Tsahi-Elkayam/cloudview/cmd/cloudview"
	"github.com/Tsahi-Elkayam/cloudview/pkg/utils"
)

func main() {
	// Initialize logger
	logger := utils.NewLogger()

	// Create and execute root command
	rootCmd := cloudview.NewRootCommand(logger)

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
		os.Exit(1)
	}
}
