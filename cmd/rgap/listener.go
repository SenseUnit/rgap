package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Snawoot/rgap"
	"github.com/Snawoot/rgap/config"
)

var (
	configPath string
)

// listenerCmd represents the listener command
var listenerCmd = &cobra.Command{
	Use:   "listener",
	Short: "Starts listener accepting and processing announcements",
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg config.ListenerConfig
		cfgF, err := os.Open(configPath)
		if err != nil {
			return fmt.Errorf("unable to read configuration file: %w", err)
		}
		defer cfgF.Close()
		dec := yaml.NewDecoder(cfgF)
		dec.KnownFields(true)
		if err := dec.Decode(&cfg); err != nil {
			return fmt.Errorf("unable to decode configuration file: %w", err)
		}
		listener, err := rgap.NewListener(&cfg)
		if err != nil {
			return fmt.Errorf("can't initialize listener: %w", err)
		}
		return listener.Run(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(listenerCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listenerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listenerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	listenerCmd.Flags().StringVarP(&configPath, "config", "c", "rgap.yaml", "configuration file")
}
