package rgap

import (
	"context"
	"fmt"
	"log"
	"time"

	"gopkg.in/yaml.v3"
)

type StartStopper interface {
	Start() error
	Stop() error
}

type GroupConfig struct {
	ID             uint64
	PSK            *PSK
	Expire         time.Duration
	ClockSkew      time.Duration `yaml:"clock_skew"`
	ReadinessDelay time.Duration `yaml:"readiness_delay"`
}

type OutputConfig struct {
	Kind string
	Spec yaml.Node
}

type ListenerConfig struct {
	Listen  []string
	Groups  []GroupConfig
	Outputs []OutputConfig
}

type Listener struct {
	sources []StartStopper
	groups  map[uint64]*Group
	outputs []StartStopper
}

func NewListener(cfg *ListenerConfig) (*Listener, error) {
	l := &Listener{
		groups: make(map[uint64]*Group),
	}
	for i, gc := range cfg.Groups {
		g, err := GroupFromConfig(&gc)
		if err != nil {
			return nil, fmt.Errorf("unable to construct new group with index %d: %w", i, err)
		}
		l.groups[g.ID()] = g
	}
	for _, address := range cfg.Listen {
		src := NewUDPSource(address, address, l.announceCallback)
		l.sources = append(l.sources, src)
	}
	for i, oc := range cfg.Outputs {
		out, err := OutputFromConfig(&oc, l)
		if err != nil {
			return nil, fmt.Errorf("unable to construct new output with index %d: %w", i, err)
		}
		l.outputs = append(l.outputs, out)
	}
	return l, nil
}

func (l *Listener) announceCallback(label string, ann *Announcement) {
	group, ok := l.groups[ann.Data.RedundancyID]
	if !ok {
		return
	}
	if err := group.Ingest(ann); err != nil {
		log.Printf("Group %d ingestion error: %v", err)
	}
}

func (l *Listener) Run(ctx context.Context) error {
	var primeStack []StartStopper
	defer func() {
		for i := len(primeStack) - 1; i >= 0; i-- {
			if err := primeStack[i].Stop(); err != nil {
				log.Printf("shutdown error: %v", err)
			}
		}
	}()
	for _, group := range l.groups {
		if err := group.Start(); err != nil {
			return fmt.Errorf("startup error: %w", err)
		}
		primeStack = append(primeStack, group)
	}
	for _, source := range l.sources {
		if err := source.Start(); err != nil {
			return fmt.Errorf("startup error: %w", err)
		}
		primeStack = append(primeStack, source)
	}
	for _, out := range l.outputs {
		if err := out.Start(); err != nil {
			return fmt.Errorf("startup error: %w", err)
		}
		primeStack = append(primeStack, out)
	}
	log.Println("Listener is now operational.")
	<-ctx.Done()
	log.Println("Listener is shutting down.")
	return nil
}

func (l *Listener) Groups() []uint64 {
	res := make([]uint64, 0, len(l.groups))
	for gid := range l.groups {
		res = append(res, gid)
	}
	return res
}

func (l *Listener) ListGroup(id uint64) []GroupItem {
	g, ok := l.groups[id]
	if !ok {
		return nil
	}
	return g.List()
}
