package output

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Snawoot/rgap/config"
	"github.com/Snawoot/rgap/iface"
	"github.com/Snawoot/rgap/util"
)

type GroupHostMapping struct {
	Group             uint64
	Hostname          string
	FallbackAddresses []util.IPAddr `yaml:"fallback_addresses"`
}

type HostsFileConfig struct {
	Interval     time.Duration
	Filename     string
	Mappings     []GroupHostMapping
	PrependLines []string `yaml:"prepend_lines"`
	AppendLines  []string `yaml:"append_lines"`
}

type HostsFile struct {
	bridge       iface.GroupBridge
	interval     time.Duration
	filename     string
	mappings     []GroupHostMapping
	prependLines []string
	appendLines  []string
	ctx          context.Context
	ctxCancel    func()
	loopDone     chan struct{}
}

func NewHostsFile(cfg *config.OutputConfig, bridge iface.GroupBridge) (*HostsFile, error) {
	var hc HostsFileConfig
	if err := cfg.Spec.Decode(&hc); err != nil {
		return nil, fmt.Errorf("cannot unmarshal log output config: %w", err)
	}
	if hc.Interval <= 0 {
		return nil, fmt.Errorf("incorrect hosts file update interval: %v", hc.Interval)
	}
	if hc.Filename == "" {
		return nil, fmt.Errorf("filename is not specified")
	}
	return &HostsFile{
		bridge:       bridge,
		interval:     hc.Interval,
		filename:     hc.Filename,
		mappings:     hc.Mappings,
		prependLines: hc.PrependLines,
		appendLines:  hc.AppendLines,
	}, nil
}

func (o *HostsFile) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	o.ctx = ctx
	o.ctxCancel = cancel
	o.loopDone = make(chan struct{})
	go o.loop()
	log.Println("started log output plugin")
	return nil
}

func (o *HostsFile) Stop() error {
	o.ctxCancel()
	<-o.loopDone
	log.Println("stopped log output plugin")
	return nil
}

func (o *HostsFile) loop() {
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

func (o *HostsFile) dump() {
	// TODO: write to file
}
