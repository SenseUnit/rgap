package output

import (
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"

	"github.com/Snawoot/rgap/config"
	"github.com/Snawoot/rgap/iface"
	"github.com/Snawoot/rgap/util"
)

type DNSMapping struct {
	Group             uint64
	FallbackAddresses []util.IPAddr `yaml:"fallback_addresses"`
}

type DNSServerConfig struct {
	BindAddress string `yaml:"bind_address"`
	Mappings    map[string]DNSMapping
}

type DNSServer struct {
	bridge      iface.GroupBridge
	bindAddress string
	mappings    map[string]DNSMapping
	tcpServer   *dns.Server
	udpServer   *dns.Server
	tcpDone     chan struct{}
	udpDone     chan struct{}
}

func NewDNSServer(cfg *config.OutputConfig, bridge iface.GroupBridge) (*DNSServer, error) {
	var oc DNSServerConfig
	if err := util.CheckedUnmarshal(&cfg.Spec, &oc); err != nil {
		return nil, fmt.Errorf("cannot unmarshal DNS output config: %w", err)
	}
	mappings := make(map[string]DNSMapping)
	for name, mapping := range oc.Mappings {
		name = strings.ToLower(strings.TrimRight(name, "."))
		mappings[name] = mapping
	}
	return &DNSServer{
		bridge:      bridge,
		bindAddress: oc.BindAddress,
		mappings:    mappings,
	}, nil
}

func (o *DNSServer) Start() error {
	var startupErr error
	o.tcpDone = make(chan struct{})
	o.udpDone = make(chan struct{})
	tcpStartupDone := make(chan struct{})
	udpStartupDone := make(chan struct{})
	o.tcpServer = &dns.Server{
		Addr:              o.bindAddress,
		Net:               "tcp",
		Handler:           o,
		UDPSize:           65536,
		NotifyStartedFunc: func() { close(tcpStartupDone) },
	}
	o.udpServer = &dns.Server{
		Addr:              o.bindAddress,
		Net:               "udp",
		Handler:           o,
		UDPSize:           65536,
		NotifyStartedFunc: func() { close(udpStartupDone) },
	}
	go func() {
		defer close(o.tcpDone)
		startupErr = o.tcpServer.ListenAndServe()
	}()
	select {
	case <-tcpStartupDone:
	case <-o.tcpDone:
		return fmt.Errorf("output DNS server (TCP) startup failed: %w", startupErr)
	}
	go func() {
		defer close(o.udpDone)
		startupErr = o.udpServer.ListenAndServe()
	}()
	select {
	case <-udpStartupDone:
	case <-o.udpDone:
		err := startupErr
		o.tcpServer.Shutdown()
		return fmt.Errorf("output DNS server (UDP) startup failed: %w", err)
	}
	log.Printf("started DNS server (%s) output plugin", o.bindAddress)
	return nil
}

func (o *DNSServer) Stop() error {
	o.udpServer.Shutdown()
	o.tcpServer.Shutdown()
	log.Printf("stopped DNS server (%s) output plugin", o.bindAddress)
	return nil
}

func (o *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
}
