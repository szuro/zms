# Zabbix Metric Shipper
Make data exported by Zabbix fly!

This program is designed to parse files created by Zabbix export, filter values based on tags and send them to a configured destination.

Features:
- Autodiscovery od export files (requires read perms to zabbix_server.conf file)
- Global tag filters
- Tag filters per target
- Internal Prometheus metrics
- Configurable buffer

# CLI arguments

`-c` - Path to zms config file<br>
`-v` - Show version info

# Configurarion

A sample configuration file may be found bellow:

```
server_config: /etc/zabbix/zabbix_server.conf
buffer_size: 100
tag_filters:
  accepted:
  - tag: <name>
    value: <value>
  rejected:
  - tag: <name>
    value: <value>
targets:
- name: <unique_name>
  type: pushgateway|gcp_cloud_monitor|azuretable|print
  connection: <connectionstring>
  tag_filters:
    accepted:
    - tag: <name>
      value: <value>
    rejected:
    - tag: <name>
      value: <value>
  source:
  - history
  - trends
  - events
```

The parameters have the following meaning:

## server_config

Absolute path to Zabbix Server config. Must be readable by ZMS. It is used to get the number of DBSyncers running and export configuration, thus getting the number of export files and their paths.

## buffer_size

Size of local in-memory buffer. It is shared between targets. Setting buffer to N will force ZMS to send N values one batch request if possible (not all targets support this).

## tag_filters

Optional filtering. May be useful when presented with a significant amount of data. No filter means every value is accepted and sent to configured targets.
If specifying accepted or rejected tags, the following logic is used:
- only accepted are provided -> only matching tags are allowed
- only rejected are specified -> everything is allowed expect for matching tags
- both accepted and rejected are provided -> only accepted tags that were not rejected later are accepted

Tag names and values _must_ be exact, currently regex or wildards are not supported.

## targets

This describes the location to send data to.
Currently only History and Trends exports are supported (and not in all targets). This will change in the future.

### name

A unique identifier for a target. Only used internally for bookkeeping and logging.

### type

Target type, or destination if you will.
Currently supported targets are:
- pushgateway
- gcp_cloud_monitor
- azuretable
- print

It is possible to define multiple targets with the same type, given that their names are unique.

### connection

Connection specific to the target type.

### source

Determines which type of exported data should be sent to this target. ZMS can only sent what's exported by Zabbix.
If there's a mismatch, there will be an error.
Note that it is possible to send different sources to different targets.

# Target overview

Here's an overwiev of what's supported for each target along with the meaning of `connection`.

| Target            | History | Trends | Events | Connection                                                                                       |
| ----------------- | ------- | ------ | ------ | ------------------------------------------------------------------------------------------------ |
| azuretable        | yes     | no     | no     | Storage account SAS URL.                                                                         |
| gcp_cloud_monitor | yes     | no     | no     | Absolute path to file with access credentials. If empty, GOOGLE_APPLICATION_CREDENTIALS is used. |
| print             | yes     | yes    | no     | stdout/stderr.                                                                                   |
| pushgateway       | yes     | no     | no     | URL of Pushgateway. May contain user and password.                                               |

# Running

It is fairly simple to run ZMS. Simply run `zmsd -c /etc/zmsd.yaml`. Of course the config file should exist.

For your convenience, a sample systemd service file is included in this repository: `zmsd.service`.

# Building

To build ZMS from source, you can use the included build PowerShell scrip.

`$ build.ps1`

When on linux you can use this oneliner:

# Contributing

All supported targets can be found in the `observer` directory.
To add your own, simply create a struct that satisfies the criteria:
- Embeds `baseObserver`
- Implements `Observer` interface
