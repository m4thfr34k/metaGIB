/*
Copyright © 2023 Daniel Charpentier <Daniel.Charpentier@gmail.com>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "metaGIB",
	Short: "NFT metadata relates operations",
	Long: `Using a user provided mint list metaGIB gibs metadata, and optionally images, directly from the blockchain 
and linked metadata file. metaGIB does NOT utilize databases, with cached and 
potentially stale/inaccurate, to provide results.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.metaGIB.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.

}
