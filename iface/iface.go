package iface

import (
	"net/netip"
	"time"
)

type GroupBridge interface {
	Groups() []uint64
	ListGroup(uint64) []GroupItem
	GroupReady(uint64) bool
}

type GroupItem interface {
	Address() netip.Addr
	ExpiresAt() time.Time
}

type StartStopper interface {
	Start() error
	Stop() error
}
