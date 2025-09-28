# Google Cloud Monitoring Plugin

A ZMS observer plugin that sends Zabbix export data to Google Cloud Monitoring as custom metrics.

## Features

- Creates custom metrics in Google Cloud Monitoring
- Supports history data (FLOAT and UNSIGNED value types only)
- Automatic metric descriptor creation
- Batch processing for efficiency
- Supports tag-based filtering
- Prometheus metrics integration

## Configuration

Add to your ZMS configuration:

```yaml
targets:
  - name: "gcp-monitoring"
    type: "gcp_cloud_monitor"
    connection: ""  # Uses default credentials
    options:
      credentials_file: "/path/to/service-account.json"  # Optional
    exports:
      - "history"
```

## Authentication

The plugin uses Google Application Default Credentials in this order:
1. Service account key file (if `credentials_file` option is provided)
2. `GOOGLE_APPLICATION_CREDENTIALS` environment variable
3. Google Cloud SDK credentials
4. Compute Engine metadata service (when running on GCE)

## Metrics Created

### History Metrics
- **Type**: `custom.googleapis.com/zabbix_export/history`
- **Kind**: Gauge
- **Value Type**: Double
- **Labels**:
  - `item`: Zabbix item name
  - `itemid`: Zabbix item ID
  - `host`: Zabbix host name

### Resource Labels
- **Type**: `generic_task`
- **Labels**:
  - `location`: "global"
  - `namespace`: "default"
  - `job`: "Zabbix Export"
  - `task_id`: hostname

## Data Processing

- Only processes FLOAT and UNSIGNED value types
- Groups metrics by ItemID for batch processing
- Automatically creates metric descriptors on first run
- Handles API errors and provides detailed error reporting

## Building

```bash
go build -buildmode=plugin -o gcp_cloud_monitor.so ./plugins/gcp_cloud_monitor
```

## Dependencies

- `cloud.google.com/go/monitoring/apiv3/v2`
- `golang.org/x/oauth2/google`
- `google.golang.org/api/option`
- Standard Go libraries

## Permissions Required

The service account needs the following IAM roles:
- `roles/monitoring.metricWriter` - To write custom metrics
- `roles/monitoring.editor` - To create metric descriptors (one-time setup)