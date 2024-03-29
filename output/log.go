package output

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/iface"
	"github.com/SenseUnit/rgap/util"
)

type LogConfig struct {
	Interval time.Duration
}

type Log struct {
	bridge    iface.GroupBridge
	interval  time.Duration
	ctx       context.Context
	ctxCancel func()
	loopDone  chan struct{}
}

func NewLog(cfg *config.OutputConfig, bridge iface.GroupBridge) (*Log, error) {
	var lc LogConfig
	if err := util.CheckedUnmarshal(&cfg.Spec, &lc); err != nil {
		return nil, fmt.Errorf("cannot unmarshal log output config: %w", err)
	}
	if lc.Interval <= 0 {
		return nil, fmt.Errorf("incorrect log interval: %v", lc.Interval)
	}
	return &Log{
		interval: lc.Interval,
		bridge:   bridge,
	}, nil
}

func (o *Log) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	o.ctx = ctx
	o.ctxCancel = cancel
	o.loopDone = make(chan struct{})
	go o.loop()
	log.Println("started log output plugin")
	return nil
}

func (o *Log) Stop() error {
	o.ctxCancel()
	<-o.loopDone
	log.Println("stopped log output plugin")
	return nil
}

func (o *Log) loop() {
	defer close(o.loopDone)
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.dump()
		}
	}
}

var readinessLabels = map[bool]string{
	true:  "READY",
	false: "NOT READY",
}

func (o *Log) dump() {
	var report strings.Builder
	fmt.Fprintln(&report, "Groups snapshot:")
	for _, gid := range o.bridge.Groups() {
		grpItems := o.bridge.ListGroup(gid)
		fmt.Fprintf(&report, "  - Group %d (%s, %d entries):\n", gid, readinessLabels[o.bridge.GroupReady(gid)], len(grpItems))
		for _, item := range grpItems {
			fmt.Fprintf(&report, "    - %s (till %v)\n", item.Address().Unmap().String(), item.ExpiresAt())
		}
	}
	log.Println(report.String())
}
