package util

import (
	"net/netip"

	"gopkg.in/yaml.v3"
)

type IPAddr netip.Addr

func (a *IPAddr) Addr() netip.Addr {
	return netip.Addr(*a)
}

func (a *IPAddr) String() string {
	return (*netip.Addr)(a).String()
}

func (a *IPAddr) MarshalYAML() (interface{}, error) {
	return a.String(), nil
}

func (a *IPAddr) UnmarshalYAML(value *yaml.Node) error {
	var decodedVal string
	if err := value.Decode(&decodedVal); err != nil {
		return err
	}
	parsedAddr, err := netip.ParseAddr(decodedVal)
	if err != nil {
		return err
	}
	*a = IPAddr(parsedAddr)
	return nil
}

func Must[V any](value V, err error) V {
	if err != nil {
		panic(err)
	}
	return value
}
