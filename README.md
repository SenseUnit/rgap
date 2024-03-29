rgap
====

Redundancy Group Announcement Protocol

RGAP allows one group of hosts to be aware about IP addresses of another group of hosts.

It is useful to inform load balancers about ever-changing IP addresses of worker servers.

Announcements are propagated via short HMAC-signed UDP messages, using unicast or multicast.

This implementation defines two primary parts: *agent* and *listener*.

Agent periodically sends unicast or broadcast UDP message to announce it's presense in particular redundancy group.

Listener accepts announces, verifies them and maintains the list of active IP addresses for each redundancy group. At the same time it exposes current list of IP addresses through its output plugins.

## Usage

### Agent example

```sh
RGAP_ADDRESS=127.1.2.3 RGAP_PSK=8f1302643b0809279794c5cc47f236561d7442b85d748bd7d1a58adfbe9ff431 rgap agent -g 1000 -i 5s
```

where RGAP\_ADDRESS is actual IP address which node exposes to the redundancy group.

### Listener

```sh
rgap listener -c /etc/rgap.yaml
```

Configuration example:

```yaml
listen:
  - 239.82.71.65:8271 # or "239.82.71.65:8271@eth0" or "239.82.71.65:8271@192.168.0.0/16"
  - 127.0.0.1:8282

groups:
  - id: 1000
    psk: 8f1302643b0809279794c5cc47f236561d7442b85d748bd7d1a58adfbe9ff431
    expire: 15s
    clock_skew: 10s
    readiness_delay: 15s

outputs:
  - kind: noop
    spec:
  - kind: log
    spec:
      interval: 60s
  - kind: eventlog
    spec: # or skip spec at all
      only_groups: # or specify null for all groups
        - 1000
  - kind: hostsfile
    spec:
      interval: 5s
      filename: hosts
      mappings:
        - group: 1000
          hostname: worker
          fallback_addresses:
            - 1.2.3.4
            - 5.6.7.8
      prepend_lines:
        - "# Auto-generated hosts file"
        - "# Do not edit manually, changes will be overwritten by RGAP"
      append_lines:
        - "# End of auto-generated file"
  - kind: dns
    spec:
      bind_address: :8253
      mappings:
        worker.example.com:
          group: 1000
          fallback_addresses:
            - 1.2.3.4
            - 5.6.7.8
        worker.example.org:
          group: 1000
          fallback_addresses:
            - 1.2.3.4
            - 5.6.7.8

```

### PSK Generator

```sh
rgap genpsk
```

## Synopsys

See `rgap help` for details of command line interface.
