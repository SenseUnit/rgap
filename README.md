rgap
====

Redundancy Group Announcement Protocol

Design notes: https://gist.github.com/Snawoot/39282757e5f7db40632e5e01280b683f

RGAP allows one group of hosts to be aware about IP addresses of other group of hosts.

It is useful to inform load balancers about ever-changing IP addresses of worker server.

Announcements are propagated via short HMAC-signed UDP messages, using unicast or multicast.

This implementation defines two primary parts: *agent* and *listener*.

Agent periodically sends unicast or broadcast UDP message to announce it's presense in particular redundancy group.

Listener accepts announces, verifies them and maintains the list of active IP addresses for each redundancy group. At the same time it exposes current list of IP addresses through its output plugins.

## Usage

### Agent example

```sh
RGAP_ADDRESS=127.1.2.3 RGAP_PSK=8f1302643b0809279794c5cc47f236561d7442b85d748bd7d1a58adfbe9ff431 rgap agent -g 1000 -i 5s
```

where RGAP\_ADDRESS is actual IP address which node exposes to redundancy group.

### Listener

```sh
rgap listener -c /etc/rgap.yaml
```

Configuration example:

```yaml
listen:
  - 239.82.71.65:8271
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
      interval: 1s
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
```

### PSK Generator

```sh
rgap genpsk
```

## Synopsys

See `rgap help` for details of command line interface.
