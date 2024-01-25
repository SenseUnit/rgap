package util

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"strings"

	"golang.org/x/exp/constraints"
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

func CheckedUnmarshal(doc *yaml.Node, dst interface{}) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("unable to re-marshal node: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("unable to re-marshal node: close failed: %w", err)
	}
	dec := yaml.NewDecoder(&buf)
	dec.KnownFields(true) // that's whole point of such marshaling round trip
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("unable to unmarshal node: %w", err)
	}
	return nil
}

func SplitAndResolveAddrSpec(spec string) (string, *net.Interface, error) {
	addrSpec, ifaceSpec, found := strings.Cut(spec, "@")
	if !found {
		return addrSpec, nil, nil
	}
	iface, err := ResolveInterface(ifaceSpec)
	if err != nil {
		return addrSpec, nil, fmt.Errorf("unable to resolve interface spec %q: %w", ifaceSpec, err)
	}
	return addrSpec, iface, nil
}

func ResolveInterface(spec string) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to enumerate interfaces: %w", err)
	}
	if pfx, err := netip.ParsePrefix(spec); err == nil {
		// look for address
		for i := range ifaces {
			addrs, err := ifaces[i].Addrs()
			if err != nil {
				// may be a problem with some interface,
				// but we still probably can find the right one
				log.Printf("WARNING: interface %s is failing to report its addresses: %v", ifaces[i].Name, err)
				continue
			}
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok {
					return nil, fmt.Errorf("unexpected type returned as address interface: %T", addr)
				}
				netipAddr, ok := netip.AddrFromSlice(ipnet.IP)
				if !ok {
					return nil, fmt.Errorf("interface %v has invalid address %s", ifaces[i].Name, ipnet.IP)
				}
				netipAddr = netipAddr.Unmap()
				if pfx.Contains(netipAddr) {
					res := ifaces[i]
					return &res, nil
				}
			}
		}
	} else {
		// look for iface name
		for i := range ifaces {
			if ifaces[i].Name == spec {
				res := ifaces[i]
				return &res, nil
			}
		}
	}
	return nil, errors.New("specified interface not found")
}

func Max[T constraints.Ordered](x, y T) T {
	if x >= y {
		return x
	}
	return y
}

func Min[T constraints.Ordered](x, y T) T {
	if x <= y {
		return x
	}
	return y
}
