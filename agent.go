package rgap

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
)

type AgentConfig struct {
	Group        uint64
	Address      netip.Addr
	Key          PSK
	Interval     time.Duration
	Destinations []string
	Dialer       Dialer
}

type Agent struct {
	cfg *AgentConfig
}

func NewAgent(cfg *AgentConfig) *Agent {
	a := &Agent{
		cfg: cfg,
	}
	if a.cfg.Dialer == nil {
		a.cfg.Dialer = new(net.Dialer)
	}
	return a
}

func (a *Agent) Run(ctx context.Context) error {
	if a.cfg.Interval <= 0 {
		return a.singleRun(ctx, time.Now())
	}

	shoot := func(t time.Time) {
		err := a.singleRun(ctx, t)
		if err != nil {
			log.Printf("run error: %v", err)
		}
	}

	ticker := time.NewTicker(a.cfg.Interval)
	defer ticker.Stop()
	shoot(time.Now())
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			shoot(t)
		}
	}
}

func (a *Agent) singleRun(ctx context.Context, t time.Time) error {
	announcement := Announcement{
		Data: AnnouncementData{
			Version:          V1,
			RedundancyID:     a.cfg.Group,
			Timestamp:        t.UnixMicro(),
			AnnouncedAddress: a.cfg.Address.As16(),
		},
	}
	sig, err := announcement.Data.CalculateSignature(a.cfg.Key)
	if err != nil {
		return fmt.Errorf("can't sign announcement %#v: %w", announcement, err)
	}
	announcement.Signature = sig
	msg, err := announcement.MarshalBinary()
	if err != nil {
		return fmt.Errorf("can't marshal announcement %#v: %w", announcement, err)
	}
	var wg sync.WaitGroup
	errors := make([]error, len(a.cfg.Destinations))
	for i, dst := range a.cfg.Destinations {
		wg.Add(1)
		go func(i int, dst string) {
			defer wg.Done()
			errors[i] = a.sendSingle(ctx, msg, dst)
		}(i, dst)
	}
	wg.Wait()
	var resErr error
	for _, err := range errors {
		if err != nil {
			resErr = multierror.Append(resErr, err)
		}
	}
	return resErr
}

func (a *Agent) sendSingle(ctx context.Context, msg []byte, dst string) error {
	conn, err := a.cfg.Dialer.DialContext(ctx, "udp", dst)
	if err != nil {
		return fmt.Errorf("Agent.sendSingle dial failed: %w", err)
	}
	defer conn.Close()
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("Agent.sendSingle send failed: %w", err)
	}
	return nil
}
