package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "afc",
	Short: "Arsenal FC tools",
}

func Execute() error {
	return rootCmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&footballDataToken, "football-data-token", "", "football-data.org API token (default: AFC_FDAPI_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&footballDataBaseURL, "football-data-base-url", "", "football-data.org API base URL override")
}
