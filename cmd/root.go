package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "pstore",
	Short:   "Test CLI for playstore uplaod",
	Version: "1.0",
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
