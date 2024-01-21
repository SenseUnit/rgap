package rgap

import (
	"errors"

	"github.com/Snawoot/rgap/output"
)

type OutputCtor func(*OutputConfig, GroupBridge) (StartStopper, error)

var outputVCMap = map[string]OutputCtor{
	"NoOp": func(_ *OutputConfig, _ GroupBridge) (StartStopper, error) {
		return output.NewNoOp(), nil
	},
}

type GroupBridge interface {
	Groups() []uint64
	ListGroup(uint64) []GroupItem
}

func OutputFromConfig(cfg *OutputConfig, bridge GroupBridge) (StartStopper, error) {
	ctor, ok := outputVCMap[cfg.Kind]
	if !ok {
		return nil, errors.New("unknown kind of output")
	}
	return ctor(cfg, bridge)
}
