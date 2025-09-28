# Prometheus Pushgateway Plugin

A ZMS observer plugin that pushes Zabbix export data to a Prometheus Pushgateway as metrics.

## Features

- Pushes history and trend data as Prometheus metrics
- Supports numeric value types (FLOAT and UNSIGNED)
- Configurable job name and instance grouping
- Supports tag-based filtering
- Prometheus metrics integration for monitoring plugin performance

## Configuration

Add to your ZMS configuration:

```yaml
targets:
  - name: "prometheus-push"
    type: "prometheus_pushgateway"
    connection: "http://pushgateway:9091"
    options:
      job_name: "zms_export"  # Optional, defaults to "zms_export"
    exports:
      - "history"
      - "trends"
```

## Connection String

The connection string should be the Pushgateway URL:
- Format: `http://pushgateway-host:port`
- Example: `http://localhost:9091`
- Example: `https://pushgateway.example.com`

## Options

- `job_name`: The job name for Prometheus metrics (default: "zms_export")

## Metrics Created

### History Metrics
- **Name**: `zabbix_history_value`
- **Type**: Gauge
- **Labels**:
  - `host`: Zabbix host name
  - `item`: Zabbix item name
  - `itemid`: Zabbix item ID
- **Grouping**: By instance (host)

### Trend Metrics
- **Names**:
  - `zabbix_trend_min` - Minimum value
  - `zabbix_trend_max` - Maximum value
  - `zabbix_trend_avg` - Average value
- **Type**: Gauge
- **Labels**:
  - `host`: Zabbix host name
  - `item`: Zabbix item name
  - `itemid`: Zabbix item ID
- **Grouping**: By instance (host)

## Data Processing

- Only processes numeric value types (FLOAT, UNSIGNED, and numeric conversions)
- Each metric is individually pushed to the gateway
- Metrics are grouped by instance (hostname)
- Registry is managed per push to avoid conflicts

## Building

```bash
go build -buildmode=plugin -o prometheus_pushgateway.so ./plugins/prometheus_pushgateway
```

## Dependencies

- `github.com/prometheus/client_golang/prometheus`
- `github.com/prometheus/client_golang/prometheus/push`
- Standard Go libraries

## Pushgateway Setup

Ensure your Pushgateway is running and accessible:

```bash
# Run Pushgateway with Docker
docker run -d -p 9091:9091 prom/pushgateway

# Or install and run locally
pushgateway --web.listen-address=":9091"
```

## Prometheus Configuration

Configure Prometheus to scrape the Pushgateway:

```yaml
scrape_configs:
  - job_name: 'pushgateway'
    static_configs:
      - targets: ['pushgateway:9091']
```