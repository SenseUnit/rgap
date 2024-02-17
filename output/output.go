package output

import (
	"errors"

	"github.com/SenseUnit/rgap/config"
	"github.com/SenseUnit/rgap/iface"
)

type OutputCtor func(*config.OutputConfig, iface.GroupBridge) (iface.StartStopper, error)

var outputVCMap = map[string]OutputCtor{
	"noop": func(_ *config.OutputConfig, _ iface.GroupBridge) (iface.StartStopper, error) {
		return NewNoOp(), nil
	},
	"log": func(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
		return NewLog(cfg, bridge)
	},
	"hostsfile": func(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
		return NewHostsFile(cfg, bridge)
	},
	"dns": func(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
		return NewDNSServer(cfg, bridge)
	},
	"eventlog": func(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
		return NewEventLog(cfg, bridge)
	},
}

func OutputFromConfig(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
	ctor, ok := outputVCMap[cfg.Kind]
	if !ok {
		return nil, errors.New("unknown kind of output")
	}
	return ctor(cfg, bridge)
}
