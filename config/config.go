package config

import (
	"net/netip"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SenseUnit/rgap/iface"
	"github.com/SenseUnit/rgap/psk"
)

type AgentConfig struct {
	Group        uint64
	Address      netip.Addr
	Key          psk.PSK
	Interval     time.Duration
	Destinations []string
	Dialer       iface.Dialer
}

type GroupConfig struct {
	ID             uint64
	PSK            *psk.PSK
	Expire         time.Duration
	ClockSkew      time.Duration `yaml:"clock_skew"`
	ReadinessDelay time.Duration `yaml:"readiness_delay"`
}

type OutputConfig struct {
	Kind string
	Spec yaml.Node
}

type ListenerConfig struct {
	Listen  []string
	Groups  []GroupConfig
	Outputs []OutputConfig
}
