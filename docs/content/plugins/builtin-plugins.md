---
title: "Built-in Plugins"
description: "Overview of built-in ZMS observer plugins"
weight: 2
---

# Built-in Plugins

ZMS includes several built-in observer plugins for popular destinations. All plugins are standalone executables using HashiCorp's go-plugin framework.

## Available Plugins

### PostgreSQL (`psql`)

Stores Zabbix history data in PostgreSQL database.

**Use Case**: Long-term storage of Zabbix metrics in a relational database for custom queries and analysis.

**Features**:
- Saves history data to `performance.messages` table
- Connection pooling with configurable limits
- Prometheus metrics for connection stats
- Supports history export type only

**Configuration**:
```yaml
targets:
  - name: "postgres-target"
    type: "psql"
    connection: "postgres://user:password@localhost/dbname?sslmode=disable"
    options:
      max_conn: "10"          # Maximum open connections
      max_idle: "5"           # Maximum idle connections
      max_conn_time: "1h"     # Maximum connection lifetime
      max_idle_time: "30m"    # Maximum idle time
    exports:
      - "history"
```

**Table Schema**:
```sql
CREATE TABLE performance.messages (
    itemid BIGINT,
    name TEXT,
    clock BIGINT,
    ns BIGINT,
    value JSONB,
    host TEXT,
    groups TEXT[],
    tags JSONB
);
```

---

### Azure Table Storage (`azure_table`)

Stores Zabbix exports in Azure Table Storage.

**Use Case**: Cloud-native storage for Zabbix data with Azure integration, ideal for hybrid cloud environments.

**Features**:
- Saves history to `history` table
- Saves trends to `trends` table
- Uses itemID as partition key for efficient queries
- Supports history and trends export types

**Configuration**:
```yaml
targets:
  - name: "azure-target"
    type: "azure_table"
    connection: "https://myaccount.table.core.windows.net/"
    exports:
      - "history"
      - "trends"
```

**Authentication**:
- Uses Azure credential chain (environment variables, managed identity, or Azure CLI)
- Set `AZURE_STORAGE_ACCOUNT` and `AZURE_STORAGE_KEY` environment variables
- Or use managed identity when running in Azure

---

### Prometheus Remote Write (`prometheus_remote_write`)

Writes Zabbix data to Prometheus via the Remote Write protocol.

**Use Case**: Integration with Prometheus monitoring stack, enabling visualization in Grafana and alerting with Alertmanager.

**Features**:
- Converts history to Prometheus time series
- Converts trends to separate metrics (min, max, avg, count)
- Timestamp ordering for Remote Write API compliance
- Only processes numeric values (FLOAT and UNSIGNED types)
- Automatically adds Zabbix tags as Prometheus labels

**Configuration**:
```yaml
targets:
  - name: "prometheus-remote"
    type: "prometheus_remote_write"
    connection: "http://prometheus:9090/api/v1/write"
    exports:
      - "history"
      - "trends"
```

**Metric Naming**:
- History: `zabbix_history{itemid="...", name="...", host="...", ...}`
- Trends: `zabbix_trend_min`, `zabbix_trend_max`, `zabbix_trend_avg`, `zabbix_trend_count`

---

### Print (`print`)

Outputs Zabbix data to stdout or stderr in human-readable format.

**Use Case**: Debugging, testing filters, and understanding data flow through ZMS.

**Features**:
- Simple text output
- Configurable output destination (stdout/stderr)
- Displays all export types (history, trends, events)
- Useful for development and troubleshooting

**Configuration**:
```yaml
targets:
  - name: "print-target"
    type: "print"
    connection: "stdout"  # or "stderr"
    exports:
      - "history"
      - "trends"
      - "events"
```

**Output Example**:
```
History: itemid=12345, name=CPU Usage, value=45.2, host=server01
Trend: itemid=12345, clock=1700000000, min=10.5, max=95.3, avg=45.2
Event: eventid=67890, name=High CPU, severity=3, hosts=[server01]
```

---

### GCP Cloud Monitor (`gcp_cloud_monitor`)

Sends Zabbix metrics to Google Cloud Monitoring (formerly Stackdriver).

**Use Case**: Integration with Google Cloud Platform monitoring, ideal for GCP-hosted infrastructure.

**Features**:
- Creates custom metrics in GCP Cloud Monitoring
- Only processes numeric values (FLOAT and UNSIGNED types)
- Supports Application Default Credentials or service account JSON
- Automatically adds Zabbix metadata as metric labels

**Configuration**:
```yaml
targets:
  - name: "gcp-target"
    type: "gcp_cloud_monitor"
    connection: ""  # Uses default credentials
    options:
      credentials_file: "/path/to/service-account.json"  # Optional
      project_id: "my-gcp-project"  # Optional if using ADC
    exports:
      - "history"
```

**Authentication**:
- Without `credentials_file`: Uses Application Default Credentials (ADC)
- With `credentials_file`: Uses specified service account JSON key
- Service account needs `roles/monitoring.metricWriter` permission

**Metric Naming**:
- Custom metrics: `custom.googleapis.com/zabbix/{item_name}`

---

### Prometheus Pushgateway (`prometheus_pushgateway`)

Pushes Zabbix metrics to Prometheus Pushgateway for batch job monitoring.

**Use Case**: Pushing Zabbix metrics to Prometheus Pushgateway for ephemeral jobs or batch processing scenarios.

**Features**:
- Creates Prometheus gauges for history and trends
- Configurable job name for metric grouping
- Per-host instance grouping
- Only processes numeric values

**Configuration**:
```yaml
targets:
  - name: "pushgateway-target"
    type: "prometheus_pushgateway"
    connection: "http://pushgateway:9091"
    options:
      job_name: "zabbix_export"  # Optional, defaults to "zms_export"
    exports:
      - "history"
      - "trends"
```

**Metric Naming**:
- Metrics use sanitized Zabbix item names
- Instance label set to host name
- Job label set to configured `job_name`

---

## Common Configuration Options

All plugins support these common configuration fields:

### Filter Configuration

Apply tag-based or group-based filtering per plugin:

```yaml
targets:
  - name: "my-target"
    type: "plugin_type"
    filter:
      accept:
        - "environment:production"
        - "service:*"
      reject:
        - "test:true"
```

### Export Types

Specify which data types the plugin should process:

```yaml
exports:
  - "history"   # Individual metric values
  - "trends"    # Hourly aggregates (min, max, avg)
  - "events"    # Problem/recovery events
```

### Connection Strings

Connection string format varies by plugin:
- **PostgreSQL**: Standard PostgreSQL connection URI
- **Azure**: Azure Table Storage account URL
- **Prometheus**: HTTP/HTTPS endpoint URL
- **GCP**: Empty (uses ADC) or not used
- **Print**: "stdout" or "stderr"

---

## Building Custom Plugins

All built-in plugins are open source and can serve as examples for custom plugin development. See the [Plugin Development Guide](plugin-development.md) for detailed instructions.

**Plugin Source Code**: `/plugins/` directory in the ZMS repository

**Example Usage**: `/examples/plugins/` directory for minimal examples

---

## Performance Considerations

### Batch Processing

Most plugins batch process records for efficiency. Consider:
- History records are processed in batches from the input
- Trends are typically smaller volume (hourly aggregates)
- Events are low volume (only on state changes)

### Network Optimization

For cloud-based plugins (Azure, GCP, Prometheus):
- Use regional endpoints when possible
- Configure appropriate timeout values
- Monitor connection pool settings
- Enable compression when supported

### Resource Usage

Plugin resource usage varies by destination:
- **PostgreSQL**: Connection pool memory, DB server resources
- **Azure/GCP**: Network bandwidth, API quotas
- **Prometheus**: Remote Write receiver resources
- **Print**: Minimal, stdout/stderr only

---

## Troubleshooting

### Plugin Not Loading

```
Error: failed to load plugin "plugin_name"
```

**Solutions**:
1. Verify plugin executable exists in `plugins_dir`
2. Check execute permissions: `chmod +x plugin_name`
3. Ensure plugin matches ZMS architecture (amd64, arm64, etc.)

### Connection Failures

```
Error: failed to connect to destination
```

**Solutions**:
1. Verify connection string format
2. Check network connectivity to destination
3. Validate credentials/authentication
4. Review destination service health

### Data Not Appearing

**Checklist**:
1. Verify export types are enabled in config
2. Check filter configuration (accept/reject patterns)
3. Review plugin logs for errors
4. Confirm destination is receiving data (check destination logs)
5. For numeric-only plugins (Prometheus, GCP): Ensure data is numeric type

### Performance Issues

**Optimization Steps**:
1. Increase buffer size in ZMS config
2. Adjust connection pool settings (PostgreSQL)
3. Enable batching where supported
4. Monitor plugin process resources
5. Review network latency to destination

---

## Next Steps

- [Plugin Development Guide](plugin-development.md) - Create custom plugins
- [Configuration Reference](../configuration/config-file.md) - Detailed config options
- [Architecture Overview](../docs/architecture.md) - Understand plugin system architecture
