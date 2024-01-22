package output

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	atomicfile "github.com/natefinch/atomic"

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
	if err := util.CheckedUnmarshal(&cfg.Spec, &hc); err != nil {
		return nil, fmt.Errorf("cannot unmarshal log output config: %w", err)
	}
	if hc.Interval <= 0 {
		return nil, fmt.Errorf("incorrect hosts file update interval: %v", hc.Interval)
	}
	if hc.Filename == "" {
		return nil, fmt.Errorf("filename is not specified")
	}
	for i, mapping := range hc.Mappings {
		if mapping.Hostname == "" {
			return nil, fmt.Errorf("mapping with index %d has no hostname defined", i)
		}
	}
	prependLines := make([]string, 0, len(hc.PrependLines))
	for _, line := range hc.PrependLines {
		prependLines = append(prependLines, strings.TrimRight(line, "\r\n"))
	}
	appendLines := make([]string, 0, len(hc.AppendLines))
	for _, line := range hc.AppendLines {
		appendLines = append(appendLines, strings.TrimRight(line, "\r\n"))
	}
	return &HostsFile{
		bridge:       bridge,
		interval:     hc.Interval,
		filename:     hc.Filename,
		mappings:     hc.Mappings,
		prependLines: prependLines,
		appendLines:  appendLines,
	}, nil
}

func (o *HostsFile) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	o.ctx = ctx
	o.ctxCancel = cancel
	o.loopDone = make(chan struct{})
	go o.loop()
	log.Printf("started hostsfile (%s) output plugin", o.filename)
	return nil
}

func (o *HostsFile) Stop() error {
	o.ctxCancel()
	<-o.loopDone
	log.Printf("stopped hostsfile (%s) output plugin", o.filename)
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
	var notReadyGroups []uint64
	for _, mapping := range o.mappings {
		if !o.bridge.GroupReady(mapping.Group) {
			notReadyGroups = append(notReadyGroups, mapping.Group)
		}
	}
	if len(notReadyGroups) > 0 {
		log.Printf("hostsfile: skipping update because following groups are not ready: %v", notReadyGroups)
		return
	}

	var buf bytes.Buffer
	for _, line := range o.prependLines {
		fmt.Fprintln(&buf, line)
	}
	for _, mapping := range o.mappings {
		items := o.bridge.ListGroup(mapping.Group)
		if len(items) == 0 {
			for _, addr := range mapping.FallbackAddresses {
				fmt.Fprintf(&buf, "%s %s\n", addr.String(), mapping.Hostname)
			}
			for _, item := range items {
				fmt.Fprintf(&buf, "%s %s\n", item.Address().Unmap().String(), mapping.Hostname)
			}
			continue
		}
	}
	for _, line := range o.appendLines {
		fmt.Fprintln(&buf, line)
	}
	log.Println(buf.String())
	if err := atomicfile.WriteFile(o.filename, &buf); err != nil {
		log.Printf("unable to update destination file: %v", err)
	}
}
