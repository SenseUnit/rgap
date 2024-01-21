package rgap

import (
	"fmt"
	"log"
	"net/netip"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type Group struct {
	id             uint64
	psk            PSK
	expire         time.Duration
	clockSkew      time.Duration
	readinessDelay time.Duration
	addrSet        *ttlcache.Cache[netip.Addr, struct{}]
	readyAt        time.Time
}

type GroupItem struct {
	Address   netip.Addr
	ExpiresAt time.Time
}

func GroupFromConfig(cfg *GroupConfig) (*Group, error) {
	if cfg.PSK == nil {
		return nil, fmt.Errorf("group %d: PSK is not set", cfg.ID)
	}
	if cfg.Expire <= 0 {
		return nil, fmt.Errorf("group %d: incorrect expiration time")
	}
	g := &Group{
		id:             cfg.ID,
		psk:            *cfg.PSK,
		expire:         cfg.Expire,
		clockSkew:      cfg.ClockSkew,
		readinessDelay: cfg.ReadinessDelay,
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
	g.readyAt = time.Now().Add(g.readinessDelay)
	log.Printf("Group %d is ready.", g.id)
	return nil
}

func (g *Group) Stop() error {
	g.addrSet.Stop()
	log.Printf("Group %d was destroyed.", g.id)
	return nil
}

func (g *Group) Ingest(a *Announcement) error {
	if a.Data.Version != V1 {
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
		g.addrSet.Set(address, struct{}{}, expireAt.Sub(now))
	}
	return nil
}

func (g *Group) List() []GroupItem {
	items := g.addrSet.Items()
	res := make([]GroupItem, 0, len(items))
	for _, item := range items {
		if item.IsExpired() {
			continue
		}
		res = append(res, GroupItem{
			Address:   item.Key(),
			ExpiresAt: item.ExpiresAt(),
		})
	}
	return res
}
