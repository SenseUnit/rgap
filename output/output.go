package output

import (
	"errors"

	"github.com/Snawoot/rgap/config"
	"github.com/Snawoot/rgap/iface"
)

type OutputCtor func(*config.OutputConfig, iface.GroupBridge) (iface.StartStopper, error)

var outputVCMap = map[string]OutputCtor{
	"noop": func(_ *config.OutputConfig, _ iface.GroupBridge) (iface.StartStopper, error) {
		return NewNoOp(), nil
	},
	"log": func(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
		return NewLog(cfg, bridge)
	},
}

func OutputFromConfig(cfg *config.OutputConfig, bridge iface.GroupBridge) (iface.StartStopper, error) {
	ctor, ok := outputVCMap[cfg.Kind]
	if !ok {
		return nil, errors.New("unknown kind of output")
	}
	return ctor(cfg, bridge)
}
