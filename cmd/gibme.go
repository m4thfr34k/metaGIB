/*
Copyright © 2023 Daniel Charpentier <Daniel.Charpentier@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var hashList string = ""
var rpcURL string = ""
var includeImages bool = false

// gibmeCmd represents the gibme command
var gibmeCmd = &cobra.Command{
	Use:   "gibme",
	Short: "Downloads metadata for a given mint list",
	Long:  `Downloads metadata for a given mint list.`,
	Run: func(cmd *cobra.Command, args []string) {
		if hashList != "" && rpcURL != "" {
			err := getGenericMetadata(hashList, rpcURL, includeImages)
			if err != nil {
				fmt.Printf("Action failed, err: %v\n", err)
			} else {
				fmt.Println("Complete")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(gibmeCmd)

	gibmeCmd.PersistentFlags().StringVarP(&hashList, "list", "l", "", "Hash list csv filename")
	gibmeCmd.PersistentFlags().StringVarP(&rpcURL, "rpc", "r", "", "RPC URL to use")
	gibmeCmd.Flags().BoolVarP(&includeImages, "images", "i", false, "Include NFT images in download")
}
