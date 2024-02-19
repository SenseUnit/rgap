package output

import (
	"fmt"
	"log"

	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/iface"
	"github.com/SenseUnit/rgap/util"
)

type EventLogConfig struct {
	Groups []uint64 `yaml:"only_groups"`
}

type EventLog struct {
	bridge   iface.GroupBridge
	groups   []uint64
	unsubFns []func()
}

func NewEventLog(cfg *config.OutputConfig, bridge iface.GroupBridge) (*EventLog, error) {
	var lc EventLogConfig
	if err := util.CheckedUnmarshal(&cfg.Spec, &lc); err != nil {
		return nil, fmt.Errorf("cannot unmarshal log output config: %w", err)
	}
	return &EventLog{
		bridge: bridge,
		groups: lc.Groups,
	}, nil
}

func (o *EventLog) Start() error {
	groups := o.groups
	if groups == nil {
		groups = o.bridge.Groups()
	}
	o.unsubFns = make([]func(), 0, len(groups)*2)
	for _, group := range groups {
		o.unsubFns = append(o.unsubFns,
			o.bridge.OnJoin(group, func(group uint64, item iface.GroupItem) {
				log.Printf("host %s has joined group %d", item.Address().Unmap().String(), group)
			}),
			o.bridge.OnLeave(group, func(group uint64, item iface.GroupItem) {
				log.Printf("host %s has left group %d", item.Address().Unmap().String(), group)
			}),
		)
	}
	log.Println("started event log output plugin")
	return nil
}

func (o *EventLog) Stop() error {
	for _, unsub := range o.unsubFns {
		unsub()
	}
	log.Println("stopped event log output plugin")
	return nil
}
