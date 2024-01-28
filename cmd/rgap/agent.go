package main

import (
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/SenseUnit/rgap/agent"
	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/psk"
)

const (
	envPSK     = "RGAP_PSK"
	envAddress = "RGAP_ADDRESS"
)

var (
	group        uint64
	address      addressOption
	key          pskOption
	interval     time.Duration
	destinations []string
)

type addressOption struct {
	addr *netip.Addr
}

func (a *addressOption) String() string {
	if a == nil || a.addr == nil {
		return "<nil>"
	}
	return a.addr.String()
}

func (a *addressOption) Set(s string) error {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return err
	}
	a.addr = &addr
	return nil
}

func (a *addressOption) Type() string {
	return "ip"
}

type pskOption struct {
	psk *psk.PSK
}

func (pskOpt *pskOption) String() string {
	if pskOpt.psk == nil {
		return "<nil>"
	}
	return pskOpt.psk.String()
}

func (pskOpt *pskOption) Set(s string) error {
	newPSK := new(psk.PSK)
	if err := newPSK.FromHexString(s); err != nil {
		return err
	}
	pskOpt.psk = newPSK
	return nil
}

func (_ *pskOption) Type() string {
	return "hexstring"
}

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run agent to send announcements",
	RunE: func(cmd *cobra.Command, args []string) error {
		if address.addr == nil {
			envAddressVal, ok := os.LookupEnv(envAddress)
			if !ok {
				return fmt.Errorf("announced address is not specified neither in command line argument nor in %s environment variable", envAddress)
			}
			if err := address.Set(envAddressVal); err != nil {
				return err
			}
		}
		if key.psk == nil {
			hexpsk, ok := os.LookupEnv(envPSK)
			if !ok {
				return fmt.Errorf("PSK is not specified neither in command line argument nor in %s environment variable", envPSK)
			}
			if err := key.Set(hexpsk); err != nil {
				return err
			}
		}
		cfg := &config.AgentConfig{
			Group:        group,
			Address:      *address.addr,
			Key:          *key.psk,
			Interval:     interval,
			Destinations: destinations,
		}
		return agent.NewAgent(cfg).Run(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// agentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// agentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	agentCmd.Flags().Uint64VarP(&group, "group", "g", 0, "redundancy group")
	agentCmd.Flags().VarP(&address, "address", "a", "IP address to announce")
	agentCmd.Flags().VarP(&key, "psk", "k", "pre-shared key for announcement signature")
	agentCmd.Flags().DurationVarP(&interval, "interval", "i", 0, "announcement interval. If not specified agent sends one announce and exits")
	agentCmd.Flags().StringArrayVarP(&destinations, "dst", "d", []string{"239.82.71.65:8271"}, "announcement destination address:port. Can be specified multiple times")
}
