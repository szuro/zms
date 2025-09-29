# Log Print Plugin

A ZMS observer plugin that filters and outputs Zabbix LOG type history items to stdout or stderr.

## Features

- Filters only LOG type history items (zbxpkg.LOG)
- Outputs to stdout or stderr
- Simple text format output
- Useful for debugging and log monitoring
- Prometheus metrics integration
- Custom filter implementation example

## Configuration

Add to your ZMS configuration:

```yaml
targets:
  - name: "log-output"
    type: "log_print"
    connection: "stdout"  # or "stderr"
    exports:
      - "history"
```

## Connection String

The connection string determines the output destination:
- `"stdout"` - Output to standard output (default)
- `"stderr"` - Output to standard error

## Output Format

### History Data (LOG type only)
```
Host: hostname; Item: item_name; Time: 1234567890; Value: log message text
```

## Data Processing

- Only processes history data with type LOG (zbxpkg.LOG)
- Custom LogFilter filters out non-LOG history items
- Trends and events are not supported (returns empty arrays)
- Updates Prometheus metrics for sent/failed operations
- Each log entry is output as a single line

## Building

```bash
go build -buildmode=plugin -o log_print.so ./examples/plugins/log_print
```

## Use Cases

- **Log Monitoring**: Output Zabbix log items to console
- **Development**: Debug log data flow and filtering
- **Testing**: Example of custom filter implementation
- **Integration**: Simple log aggregation pipeline

## Example Output

```
Host: web01; Item: system.log; Time: 1640995200; Value: Application started successfully
Host: web01; Item: app.log; Time: 1640995200; Value: User login: admin
Host: db01; Item: database.log; Time: 1640995200; Value: Connection pool exhausted
```

## Custom Filter Implementation

This plugin demonstrates how to implement a custom filter:

- **LogFilter** struct implements the `filter.Filter` interface
- **AcceptHistory()** filters only LOG type history items
- **AcceptTrend()** and **AcceptEvent()** reject all trends and events
- **PrepareFilter()** returns the custom LogFilter instance

This approach allows fine-grained control over what data is processed by the observer.

## Dependencies

- Standard Go libraries only
- No external dependencies required