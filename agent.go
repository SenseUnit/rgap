package rgap

import (
	"net/netip"
	"time"
)

type AgentConfig struct {
	Group        uint64
	Address      netip.Addr
	Key          PSK
	Interval     time.Duration
	Destinations []string
}
