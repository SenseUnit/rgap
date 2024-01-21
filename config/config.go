package config

import (
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Snawoot/rgap/psk"
)

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
