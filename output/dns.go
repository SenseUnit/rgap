package output

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/miekg/dns"
	"pgregory.net/rand"

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
	Compress    bool
}

type DNSServer struct {
	bridge      iface.GroupBridge
	bindAddress string
	mappings    map[string]DNSMapping
	compress    bool
	tcpServer   *dns.Server
	udpServer   *dns.Server
	tcpDone     chan struct{}
	udpDone     chan struct{}
	rand        *rand.Rand
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
		compress:    oc.Compress,
		rand:        rand.New(),
	}, nil
}

func (o *DNSServer) Start() error {
	var (
		tcpStartupErr error
		udpStartupErr error
	)
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
		tcpStartupErr = o.tcpServer.ListenAndServe()
	}()
	select {
	case <-tcpStartupDone:
	case <-o.tcpDone:
		return fmt.Errorf("output DNS server (TCP) startup failed: %w", tcpStartupErr)
	}
	go func() {
		defer close(o.udpDone)
		udpStartupErr = o.udpServer.ListenAndServe()
	}()
	select {
	case <-udpStartupDone:
	case <-o.udpDone:
		o.tcpServer.Shutdown()
		return fmt.Errorf("output DNS server (UDP) startup failed: %w", udpStartupErr)
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

func (o *DNSServer) failDNSReq(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.Compress = o.compress
	m.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(m)
}

func (o *DNSServer) serveEmptyResponse(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.Compress = o.compress
	m.SetReply(r)
	w.WriteMsg(m)
}

func (o *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		o.failDNSReq(w, r)
		return
	}

	dom := r.Question[0].Name
	name := strings.ToLower(strings.TrimRight(dom, "."))
	qtype := r.Question[0].Qtype

	if r.Question[0].Qclass != dns.ClassINET {
		o.failDNSReq(w, r)
		return
	}

	log.Printf("DNS req @ %s: Name = %q, QType = %s", o.bindAddress, name, dns.Type(qtype).String())

	switch qtype {
	case dns.TypeAAAA, dns.TypeA:
	default:
		o.failDNSReq(w, r)
		return
	}

	mapping, ok := o.mappings[name]
	if !ok {
		o.failDNSReq(w, r)
		return
	}

	if !o.bridge.GroupReady(mapping.Group) {
		o.failDNSReq(w, r)
		return
	}

	m := new(dns.Msg)
	m.Compress = o.compress

	items := o.bridge.ListGroup(mapping.Group)
	if len(items) == 0 {
		// group is empty - fallback needed
		for _, addr := range mapping.FallbackAddresses {
			netAddr := addr.Addr()
			switch qtype {
			case dns.TypeA:
				if netAddr.Is4() {
					m.Answer = append(m.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: dom, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   netAddr.AsSlice(),
					})
				}
			case dns.TypeAAAA:
				if netAddr.Is6() {
					m.Answer = append(m.Answer, &dns.AAAA{
						Hdr:  dns.RR_Header{Name: dom, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
						AAAA: netAddr.AsSlice(),
					})
				}
			}
		}
	} else {
		now := time.Now()
		for _, item := range items {
			netAddr := item.Address().Unmap()
			ttl := uint32(item.ExpiresAt().Sub(now).Seconds())
			switch qtype {
			case dns.TypeA:
				if netAddr.Is4() {
					m.Answer = append(m.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: dom, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
						A:   netAddr.AsSlice(),
					})
				}
			case dns.TypeAAAA:
				if netAddr.Is6() {
					m.Answer = append(m.Answer, &dns.AAAA{
						Hdr:  dns.RR_Header{Name: dom, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
						AAAA: netAddr.AsSlice(),
					})
				}
			}
		}
	}
	rand.ShuffleSlice(o.rand, m.Answer)
	m.SetReply(r)
	w.WriteMsg(m)
}
