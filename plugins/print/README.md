# Print Plugin

A ZMS observer plugin that outputs Zabbix export data to stdout or stderr for debugging and testing purposes.

## Features

- Outputs history and trend data to stdout or stderr
- Supports tag-based filtering
- Simple text format output
- Useful for debugging and development
- Prometheus metrics integration

## Configuration

Add to your ZMS configuration:

```yaml
targets:
  - name: "debug-output"
    type: "print"
    connection: "stdout"  # or "stderr"
    exports:
      - "history"
      - "trends"
```

## Connection String

The connection string determines the output destination:
- `"stdout"` - Output to standard output (default)
- `"stderr"` - Output to standard error

## Output Format

### History Data
```
Host: hostname; Item: item_name; Time: 1234567890; Value: 42.5
```

### Trend Data
```
Host: hostname; Item: item_name; Time: 1234567890; Min/Max/Avg: 10.0/50.0/30.0
```

## Data Processing

- Only processes history and trend data (events will cause a panic)
- Applies configured tag filters before output
- Updates Prometheus metrics for sent/failed operations
- Each data point is output as a single line

## Building

```bash
go build -buildmode=plugin -o print.so ./plugins/print
```

## Use Cases

- **Development**: Debug data flow and filtering
- **Testing**: Verify plugin system functionality
- **Monitoring**: Simple output for log aggregation
- **Validation**: Check data format and content

## Example Output

```
Host: web01; Item: cpu.usage; Time: 1640995200; Value: 45.2
Host: web01; Item: memory.free; Time: 1640995200; Value: 2048000000
Host: db01; Item: connections; Time: 1640995200; Min/Max/Avg: 10.0/150.0/75.5
```

## Dependencies

- Standard Go libraries only
- No external dependencies required