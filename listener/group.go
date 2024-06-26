package listener

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/iface"
	"github.com/SenseUnit/rgap/protocol"
	"github.com/SenseUnit/rgap/psk"
	"github.com/SenseUnit/rgap/util"
	"github.com/jellydator/ttlcache/v3"
)

type Group struct {
	id               uint64
	psk              psk.PSK
	expire           time.Duration
	clockSkew        time.Duration
	readinessDelay   time.Duration
	addrSet          *ttlcache.Cache[netip.Addr, struct{}]
	ready            atomic.Bool
	readinessBarrier chan struct{}
	readinessTimer   *time.Timer
}

type groupItem struct {
	address   netip.Addr
	expiresAt time.Time
}

func (gi groupItem) Address() netip.Addr {
	return gi.address
}

func (gi groupItem) ExpiresAt() time.Time {
	return gi.expiresAt
}

func GroupFromConfig(cfg *config.GroupConfig) (*Group, error) {
	if cfg.PSK == nil {
		return nil, fmt.Errorf("group %d: PSK is not set", cfg.ID)
	}
	if cfg.Expire <= 0 {
		return nil, fmt.Errorf("group %d: incorrect expiration time", cfg.Expire)
	}
	g := &Group{
		id:               cfg.ID,
		psk:              *cfg.PSK,
		expire:           cfg.Expire,
		clockSkew:        cfg.ClockSkew,
		readinessDelay:   cfg.ReadinessDelay,
		readinessBarrier: make(chan struct{}),
		addrSet: ttlcache.New[netip.Addr, struct{}](
			ttlcache.WithDisableTouchOnHit[netip.Addr, struct{}](),
		),
	}
	if g.clockSkew <= 0 {
		g.clockSkew = g.expire
	}
	if g.clockSkew > g.expire {
		// we'll cap it by expiration time anyway,
		// as well as not allow messages from distant future
		g.clockSkew = g.expire
	}
	return g, nil
}

func (g *Group) ID() uint64 {
	return g.id
}

func (g *Group) Start() error {
	go g.addrSet.Start()
	g.readinessTimer = time.AfterFunc(g.readinessDelay, func() {
		g.ready.Store(true)
		close(g.readinessBarrier)
	})
	log.Printf("Group %d was started.", g.id)
	return nil
}

func (g *Group) Stop() error {
	g.addrSet.Stop()
	if g.readinessTimer != nil {
		g.readinessTimer.Stop()
	}
	log.Printf("Group %d was destroyed.", g.id)
	return nil
}

func (g *Group) Ingest(a *protocol.Announcement) error {
	if a.Data.Version != protocol.V1 {
		return nil
	}
	now := time.Now()
	announceTime := time.UnixMicro(a.Data.Timestamp)
	timeDrift := now.Sub(announceTime)
	if timeDrift.Abs() > g.clockSkew {
		return nil
	}
	ok, err := a.CheckSignature(g.psk)
	if err != nil {
		// normally shouldn't happen. Notify user by raising this error.
		return fmt.Errorf("announce verification failed: %w", err)
	}
	if !ok {
		return nil
	}
	address := netip.AddrFrom16(a.Data.AnnouncedAddress)
	expireAt := announceTime.Add(g.expire)
	setItem := g.addrSet.Get(address)
	if setItem == nil || setItem.ExpiresAt().Before(expireAt) {
		g.addrSet.Set(address, struct{}{}, util.Max(expireAt.Sub(now), 1))
	}
	return nil
}

func (g *Group) List() []iface.GroupItem {
	items := g.addrSet.Items()
	res := make([]iface.GroupItem, 0, len(items))
	for _, item := range items {
		if item.IsExpired() {
			continue
		}
		res = append(res, groupItem{
			address:   item.Key(),
			expiresAt: item.ExpiresAt(),
		})
	}
	return res
}

func (g *Group) Ready() bool {
	return g.ready.Load()
}

func (g *Group) ReadinessBarrier() <-chan struct{} {
	return g.readinessBarrier
}

func (g *Group) OnJoin(cb iface.GroupEventCallback) func() {
	return g.addrSet.OnInsertion(func(_ context.Context, item *ttlcache.Item[netip.Addr, struct{}]) {
		cb(g.id, groupItem{
			address:   item.Key(),
			expiresAt: item.ExpiresAt(),
		})
	})
}

func (g *Group) OnLeave(cb iface.GroupEventCallback) func() {
	return g.addrSet.OnEviction(func(_ context.Context, _ ttlcache.EvictionReason, item *ttlcache.Item[netip.Addr, struct{}]) {
		cb(g.id, groupItem{
			address:   item.Key(),
			expiresAt: item.ExpiresAt(),
		})
	})
}
