package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "glx",
	Short: "GENEALOGIX CLI - Manage and validate genealogy archives",
	Long: `GLX is the official command-line tool for working with GENEALOGIX family archives.

GENEALOGIX is a modern, evidence-first, Git-native genealogy data standard.
Use GLX to initialize new archives, validate files, and ensure data quality.`,
	Version: "1.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("glx version {{.Version}}\n")
}

