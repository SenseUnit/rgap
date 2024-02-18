package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

const (
	envLogPrefix = "RGAP_LOG_PREFIX"
)

var (
	logPrefix logPrefixValue = newLogPrefixValue(defaultLogPrefix())
)

type logPrefixValue struct {
	value *string
}

func newLogPrefixValue(s string) logPrefixValue {
	return logPrefixValue{
		value: &s,
	}
}

func (v *logPrefixValue) String() string {
	if v == nil || v.value == nil {
		return defaultLogPrefix()
	}
	return *v.value
}

func (v *logPrefixValue) Type() string {
	return "string"
}

func (v *logPrefixValue) Set(s string) error {
	v.value = &s
	return nil
}

func defaultLogPrefix() string {
	if envLogPrefixValue, ok := os.LookupEnv(envLogPrefix); ok {
		return envLogPrefixValue
	}
	return strings.ToUpper(progName) + ": "
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          progName,
	Short:        "Redundancy Group Announcement Protocol",
	Long:         `See https://gist.github.com/Snawoot/39282757e5f7db40632e5e01280b683f for more details.`,
	SilenceUsage: true,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()
	log.Default().SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Default().SetPrefix(logPrefix.String())
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().Var(&logPrefix, "log-prefix", "log prefix")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
