rgap
====

Redundancy Group Announcement Protocol

RGAP allows one group of hosts to be aware about IP addresses of another group of hosts. For example, it is useful for updating load balancers on dynamic IP addresses of backend servers.

Announcements are propagated in short HMAC-signed UDP messages using unicast or multicast.

This implementation defines two primary parts: *agent* and *listener*.

Agent periodically sends unicast or broadcast UDP message to announce it's presense in particular redundancy group.

Listener accepts announces, verifies them and maintains the list of active IP addresses for each redundancy group. At the same time it exposes current list of IP addresses through its output plugins.

## Usage

### Agent example

```sh
RGAP_ADDRESS=127.1.2.3 \
RGAP_PSK=8f1302643b0809279794c5cc47f236561d7442b85d748bd7d1a58adfbe9ff431 \
    rgap agent -g 1000 -i 5s
```

where RGAP\_ADDRESS is actual IP address which node exposes to the redundancy group.

### Listener

```sh
rgap listener -c /etc/rgap.yaml
```

See also [configuration example](#configuration-example).

### PSK Generator

```sh
rgap genpsk
```

## Reference

### Listener confiruration

The file is in YAML syntax with following elements

* **`listen`** (_list_)
    * (_string_) listen port addresses. Accepted formats: _host:port_ or _host:port@interface_ or _host:port@IP/prefixlen_. In later case rgap will find an interface with IP address which belongs to network specified by _IP/prefixlen_. Examples: `239.82.71.65:8271`, `239.82.71.65:8271@eth0`, `239.82.71.65:8271@192.168.0.0/16`.
* **`groups`** (_list_)
    * (_dictionary_)
        * **`id`** (_uint64_) redundancy group identifier.
        * **`psk`** (_string_) hex-encoded pre-shared key for message authentication.
        * **`expire`** (_duration_) how long announced address considered active past the timestamp specified in the announcement.
        * **`clock_skew`** (_duration_) allowed skew between local clock and time in announcement message.
        * **`readiness_delay`** (_duration_) startup delay before group is reported as READY to output plugins. Useful to supress uninitialized group output after startup.
* **`outputs`** (_list_)
    * (_dictionary_)
        * **`kind`** (_string_) name of output plugin
        * **`spec`** (_any_) YAML config of corresponding output plugin

### Output plugins reference

#### `noop`

Dummy plugin which doesn't do anything.

#### `log`

Periodically dumps groups contents to the application log.

Configuration:

* **`interval`** (duration) interval between dumps into log.

#### `eventlog`

Logs group membership changes to the application log.

Configuration:

* **`only_groups`** (_list_ or _null_) list of group identifiers to subscribe to. All groups are logged if this list is `null` or this key is not specified.
    * (_uint64_) group ID.

#### `hostsfile`

Periodically dumps group contents into hosts file.

Configuration:

* **`interval`** (_duration_) interval between periodic dumps.
* **`filename`** (_string_) path to hosts file
* **`mappings`** (_list_)
    * (_dictionary_)
        * **`group`** (_uint64_) group which addresses should be mapped to given hostname in hosts file
        * **`hostname`** (_string_) hostname specified for group addresses in hosts file
        * **`fallback_addresses`** (_list_)
            * (_string_) addresses to use instead of group addresses if group is empty
* **`prepend_lines`** (_list_)
    * (_string_) lines to prepend before output. Useful for comment lines.
* **`append_lines`** (_list_)
    * (_string_) lines to append after output. Useful for comment lines.

#### `dns`

Runs DNS server responding to queries for names mapped to group addresses.

Configuration:

* **`bind_address`** (_string_)
* **`mappings`** (_dictionary_)
    * **\*DOMAIN NAME\*** (_dictionary_)
        * **`group`** (_uint64_) group ID which addresses whould be returned in response to DNS queries for hostname **\*DOMAIN NAME\***.
        * **`fallback_addresses`** (_list_)
            * (_string_) addresses to use instead of group addresses if group is empty
* **`compress`** (_boolean_) compress DNS response message

#### `command`

Pipes active addresses of group into stdin of external command after each membership change. Redirects stdout and stderr of external command to output into application log.

Configuration:

* **`group`** (_uint64_) identifier of group.
* **`command`** (_list of strings_) command and arguments.
* **`timeout`** (_duration_) execution time limit for the command.
* **`retries`** (_int_) attempts to retry failed command. Default is `1`.
* **`wait_delay`** (_duration_) delay to wait for I/O to complete after process termination. Zero value disables I/O cancellation logic. Default is `100ms`.

### Configuration example

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
  - kind: command
    spec:
      group: 1000
      command:
        - "/home/user/sync.sh"
        - "--group"
        - "1000"
      timeout: 5s
      retries: 3

```
 
### CLI synopsis

Run `rgap help` to see details of command line interface.
