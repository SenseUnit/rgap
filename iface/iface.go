package iface

import (
	"context"
	"net"
	"net/netip"
	"time"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

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
