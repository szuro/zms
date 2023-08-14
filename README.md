# Zabbix Metric Shipper
Make data exported by Zabbix fly!

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
  source:
  - history
  - trends
  - events
```

The parameters have the following meaning:

## server_config

Absolute path to Zabbix Server config. Must be readable by ZMS. It is used to get th enumber of DBSyncers running, thus getting the number of export files.

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

This desribes the location to send data to.
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

### connection

Connection specific to the target type.

### source

Data to sent to the target. ZMS can only sent what's exported by Zabbix.
If there's a missmatch, there will be an error.

# Target overview

Here's an overwiev of what's supported for each target along with the meaning of `connection`.

| Target            | History | Trends | Events | Connection                                                                                       |
| ----------------- | ------- | ------ | ------ | ------------------------------------------------------------------------------------------------ |
| azuretable        | yes     | no     | no     | Storage account SAS URL.                                                                         |
| gcp_cloud_monitor | yes     | no     | no     | Absolute path to file with access credentials. If empty, GOOGLE_APPLICATION_CREDENTIALS is used. |
| print             | yes     | yes    | no     | stdout/stderr.                                                                                   |
| pushgateway       | yes     | no     | no     | URL of Pushgateway. May contain user and password.                                               |
