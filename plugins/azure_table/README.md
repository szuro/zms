# Azure Table Storage Plugin

A ZMS observer plugin that stores Zabbix export data in Azure Table Storage.

## Features

- Stores history and trend data in separate Azure tables
- Uses Azure Tables client with no credential authentication
- Supports tag-based filtering
- Prometheus metrics integration

## Configuration

Add to your ZMS configuration:

```yaml
targets:
  - name: "azure-storage"
    type: "azure_table"
    connection: "https://<storage-account>.table.core.windows.net/?<SAS token>"
    exports:
      - "history"
      - "trends"
```

## Connection String

The connection string should be the Azure Table Storage service URL:
- Format: `https://<storage-account>.table.core.windows.net/?<SAS token>`
- Authentication: Uses Azure default credentials (environment variables, managed identity, etc.)

## Tables Created

- `history` - Stores Zabbix history data
- `trends` - Stores Zabbix trend data

## Data Structure

### History Table
- **PartitionKey**: ItemID
- **RowKey**: Clock.Nanoseconds
- **Data**: Complete history record with host information

### Trends Table
- **PartitionKey**: ItemID
- **RowKey**: Clock timestamp
- **Data**: Complete trend record with min/max/avg values

## Building

```bash
go build -buildmode=plugin -o azure_table.so ./plugins/azure_table
```

## Dependencies

- `github.com/Azure/azure-sdk-for-go/sdk/data/aztables`
- Standard Go libraries