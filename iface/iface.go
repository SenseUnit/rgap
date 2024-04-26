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

type GroupEventCallback = func(group uint64, item GroupItem)

type GroupBridge interface {
	Groups() []uint64
	ListGroup(uint64) []GroupItem
	GroupReady(uint64) bool
	GroupReadinessBarrier(uint64) <-chan struct{}
	OnJoin(uint64, GroupEventCallback) func()
	OnLeave(uint64, GroupEventCallback) func()
}

type GroupItem interface {
	Address() netip.Addr
	ExpiresAt() time.Time
}

type StartStopper interface {
	Start() error
	Stop() error
}
