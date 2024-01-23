package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/Snawoot/rgap/config"
	"github.com/Snawoot/rgap/protocol"
	"github.com/Snawoot/rgap/util"
	"github.com/hashicorp/go-multierror"
)

type Agent struct {
	cfg *config.AgentConfig
}

func NewAgent(cfg *config.AgentConfig) *Agent {
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
		runCtx, done := context.WithTimeout(ctx, a.cfg.Interval)
		defer done()
		err := a.singleRun(runCtx, t)
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
	announcement := protocol.Announcement{
		Data: protocol.AnnouncementData{
			Version:          protocol.V1,
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
	dstAddr, iface, err := util.SplitAndResolveAddrSpec(dst)
	if err != nil {
		return fmt.Errorf("destination %s: interface resolving failed: %w", dst, err)
	}

	conn, err := a.dialInterfaceContext(ctx, "udp", dstAddr, iface)
	if err != nil {
		return fmt.Errorf("Agent.sendSingle dial failed: %w", err)
	}
	connCloseSignal := make(chan struct{})
	defer close(connCloseSignal)
	go func() {
		select {
		case <-connCloseSignal:
			conn.Close()
		case <-ctx.Done():
			conn.Close()
		}
	}()
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("Agent.sendSingle send failed: %w", err)
	}
	return nil
}

func (a *Agent) dialInterfaceContext(ctx context.Context, network, addr string, iif *net.Interface) (net.Conn, error) {
	if iif == nil {
		return a.cfg.Dialer.DialContext(ctx, network, addr)
	}

	var hints []string
	addrs, err := iif.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			return nil, fmt.Errorf("unexpected type returned as address interface: %T", addr)
		}
		netipAddr, ok := netip.AddrFromSlice(ipnet.IP)
		if !ok {
			return nil, fmt.Errorf("interface %v has invalid address %s", iif.Name, ipnet.IP)
		}
		hints = append(hints, netipAddr.Unmap().String())
	}
	boundDialer := util.NewBoundDialer(a.cfg.Dialer, strings.Join(hints, ","))
	return boundDialer.DialContext(ctx, network, addr)
}
