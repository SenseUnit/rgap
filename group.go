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
		addrSet:        ttlcache.New[netip.Addr, struct{}](),
	}
	if g.clockSkew <= 0 {
		g.clockSkew = g.expire
	}
	return g, nil
}

func (g *Group) ID() uint64 {
	return g.id
}

func (g *Group) PSK() PSK {
	return g.psk
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
