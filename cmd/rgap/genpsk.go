package main

import (
	"fmt"

	"github.com/Snawoot/rgap/psk"
	"github.com/spf13/cobra"
)

// genpskCmd represents the genpsk command
var genpskCmd = &cobra.Command{
	Use:   "genpsk",
	Short: "Generate and output hex-encoded pre-shared key",
	RunE: func(cmd *cobra.Command, args []string) error {
		psk, err := psk.GeneratePSK()
		if err != nil {
			return fmt.Errorf("PSK generation failed: %w", err)
		}
		fmt.Println(psk.String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(genpskCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// genpskCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// genpskCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
