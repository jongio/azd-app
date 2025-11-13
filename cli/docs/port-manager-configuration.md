# Port Manager Configuration

The port manager provides intelligent port allocation for services with several configurable options.

## Environment Variables

### Port Range Configuration

You can customize the port range used by the port manager:

- **`AZD_PORT_RANGE_START`**: Starting port for allocation (default: 3000)
- **`AZD_PORT_RANGE_END`**: Ending port for allocation (default: 65535)

**Example:**
```bash
# Use ports 5000-8000 only
export AZD_PORT_RANGE_START=5000
export AZD_PORT_RANGE_END=8000
azd run
```

**Validation:**
- Port values must be between 1-65535
- Invalid values fall back to defaults with a warning
- Structured logging shows when custom ranges are used

### Default Ranges

- **Minimum (3000)**: Avoids well-known ports (0-1023) and registered ports (1024-2999) which often require admin privileges
- **Maximum (65535)**: Standard TCP/IP port limit

## Cache Management

### LRU Cache

The port manager implements a Least Recently Used (LRU) cache to prevent memory leaks in long-running processes:

- **Maximum cache size**: 50 port managers
- **Eviction policy**: Oldest (least recently used) entries are removed when cache is full
- **Tracking**: Each cache access updates the last-used timestamp

### Cache Behavior

- Port managers are cached per project directory (absolute path)
- Accessing a cached manager updates its last-used time
- When the cache reaches 50 entries, the least recently used manager is evicted
- Cache eviction is logged with structured logging for debugging

## Structured Logging

The port manager uses Go's `log/slog` for structured logging:

### Log Levels

- **DEBUG**: Port availability checks, cache hits/misses, path resolution
- **INFO**: Custom port range configuration detected
- **WARN**: Configuration errors, failed operations
- **ERROR**: Critical failures (e.g., path resolution)

### Example Debug Output

```
2025/11/12 23:51:53 DEBUG getting port manager path=/my/project normalized=/abs/path/to/project
2025/11/12 23:51:53 DEBUG returning cached port manager path=/abs/path/to/project
2025/11/12 23:51:53 DEBUG checking assigned port service=frontend port=3000
2025/11/12 23:51:53 DEBUG port is available port=3000
```

### Example Custom Configuration

```
2025/11/12 23:51:53 INFO using custom port range start port=5000
2025/11/12 23:51:53 INFO using custom port range end port=8000
```

## Performance Improvements

### Port Scanning

- **Bounded scanning**: Maximum 100 port attempts instead of scanning entire range
- **Early termination**: Stops on first available port
- **Error message**: Clear indication when no ports found after 100 attempts

### Error Handling

All save operations now return errors instead of logging warnings:
- Prevents silent data loss
- Enables proper error propagation
- Callers can handle failures appropriately

## TOCTOU Considerations

**Note**: There is a potential Time-Of-Check-Time-Of-Use (TOCTOU) race condition between checking port availability and binding to it. Another process could bind to the port in between.

**Mitigation**: Callers should handle port binding failures gracefully and may trigger port reassignment on binding errors.

## Best Practices

1. **Set port ranges** for containerized/isolated environments to avoid conflicts
2. **Monitor logs** at DEBUG level to understand port allocation patterns
3. **Handle binding errors** in your service startup code
4. **Use explicit ports** in azure.yaml for production services
5. **Clean stale assignments** regularly in long-running environments

## Examples

### Development with Custom Range

```bash
# Development environment - use high ports to avoid conflicts
export AZD_PORT_RANGE_START=8000
export AZD_PORT_RANGE_END=9000
azd run
```

### CI/CD Environment

```bash
# CI environment - use isolated range
export AZD_PORT_RANGE_START=30000
export AZD_PORT_RANGE_END=31000
azd run
```

### Production

```yaml
# azure.yaml - explicit ports for production
services:
  frontend:
    project: ./frontend
    port: 3000  # Explicit, repeatable port
  
  backend:
    project: ./backend
    port: 8080  # Explicit, repeatable port
```
