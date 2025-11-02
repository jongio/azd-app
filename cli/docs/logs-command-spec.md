# Logs Command Specification

## Overview

The `logs` command allows developers to view real-time and historical output logs from running services for debugging purposes. It integrates with the dashboard to provide both CLI and web-based log viewing.

## Command Interface

### Basic Usage

```bash
azd app logs [service-name] [flags]
```

### Flags

- `-f, --follow`: Follow log output (tail -f behavior, default: true for single service)
- `-s, --service <name>`: Filter logs by service name (can specify multiple with comma-separated list)
- `-n, --tail <number>`: Number of lines to show from the end (default: 100)
- `--since <duration>`: Show logs since duration (e.g., "5m", "1h", default: all)
- `--timestamps`: Show timestamps with each log entry (default: true)
- `--no-color`: Disable colored output
- `--level <level>`: Filter by log level (info, warn, error, debug, all) (default: all)
- `--format <format>`: Output format (text, json) (default: text)
- `--output <file>`: Write logs to file instead of stdout

### Examples

```bash
# View logs from all services (live tail)
azd app logs

# View logs from a specific service
azd app logs api

# View logs from multiple services
azd app logs -s api,frontend

# View last 50 lines without following
azd app logs -n 50 --follow=false

# View logs from last 5 minutes
azd app logs --since 5m

# Export logs to file
azd app logs --output logs.txt

# View only error logs
azd app logs --level error

# JSON output for parsing
azd app logs --format json
```

## Architecture

### Log Collection

Services already have `Stdout` and `Stderr` pipes created in `service.StartService()`. We need to:

1. **Create a log buffer** for each service that stores recent log entries
2. **Stream logs** to:
   - In-memory circular buffer (for quick retrieval)
   - Optional disk-based log files (`.azure/logs/<service-name>.log`)
   - WebSocket clients (dashboard)
   - CLI followers (via `azd app logs`)

### Data Flow

```
Service Process
    ↓ (stdout/stderr pipes)
Log Collector (goroutine)
    ↓
├─→ Circular Buffer (in-memory, last N lines)
├─→ Log File (optional, `.azure/logs/<service>.log`)
├─→ WebSocket Broadcast (dashboard clients)
└─→ CLI Followers (active `azd app logs` commands)
```

### Components

#### 1. Log Buffer (`internal/service/logbuffer.go`)

```go
type LogBuffer struct {
    serviceName string
    entries     []LogEntry
    maxSize     int
    mu          sync.RWMutex
    subscribers []chan LogEntry
    filePath    string // Optional file logging
}

// Add appends a log entry to the buffer
func (lb *LogBuffer) Add(entry LogEntry)

// GetRecent returns the last N entries
func (lb *LogBuffer) GetRecent(n int) []LogEntry

// GetSince returns entries since a specific time
func (lb *LogBuffer) GetSince(since time.Time) []LogEntry

// Subscribe returns a channel for live log streaming
func (lb *LogBuffer) Subscribe() <-chan LogEntry

// Unsubscribe removes a subscriber
func (lb *LogBuffer) Unsubscribe(ch <-chan LogEntry)

// Clear empties the buffer
func (lb *LogBuffer) Clear()
```

#### 2. Log Manager (`internal/service/logmanager.go`)

Manages log buffers for all services in a project.

```go
type LogManager struct {
    projectDir string
    buffers    map[string]*LogBuffer // key: serviceName
    mu         sync.RWMutex
}

// GetManager returns the log manager for a project
func GetManager(projectDir string) *LogManager

// CreateBuffer creates a log buffer for a service
func (lm *LogManager) CreateBuffer(serviceName string, maxSize int, enableFileLogging bool) *LogBuffer

// GetBuffer retrieves a log buffer
func (lm *LogManager) GetBuffer(serviceName string) (*LogBuffer, bool)

// GetAllLogs returns logs from all services
func (lm *LogManager) GetAllLogs(n int) []LogEntry

// Clear removes all log buffers
func (lm *LogManager) Clear()
```

#### 3. Log Collector Integration

Update `service.StartService()` to create log collector goroutines:

```go
// In executor.go after starting process
go collectLogs(process, logManager)

func collectLogs(process *ServiceProcess, logManager *LogManager) {
    buffer := logManager.CreateBuffer(process.Name, 1000, true)
    
    // Collect stdout
    go readAndBuffer(process.Stdout, process.Name, buffer, false)
    
    // Collect stderr
    go readAndBuffer(process.Stderr, process.Name, buffer, true)
}

func readAndBuffer(reader io.ReadCloser, serviceName string, buffer *LogBuffer, isStderr bool) {
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        entry := LogEntry{
            Service:   serviceName,
            Message:   scanner.Text(),
            Timestamp: time.Now(),
            IsStderr:  isStderr,
            Level:     inferLogLevel(scanner.Text()),
        }
        buffer.Add(entry)
    }
}
```

#### 4. Logs Command (`cmd/app/commands/logs.go`)

```go
package commands

import (
    "github.com/spf13/cobra"
)

var (
    logsFollow     bool
    logsService    string
    logsTail       int
    logsSince      string
    logsTimestamps bool
    logsNoColor    bool
    logsLevel      string
    logsFormat     string
    logsOutput     string
)

func NewLogsCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "logs [service-name]",
        Short: "View logs from running services",
        Long:  `Display output logs from running services for debugging and monitoring`,
        RunE:  runLogs,
    }
    
    cmd.Flags().BoolVarP(&logsFollow, "follow", "f", true, "Follow log output")
    cmd.Flags().StringVarP(&logsService, "service", "s", "", "Filter by service name(s)")
    cmd.Flags().IntVarP(&logsTail, "tail", "n", 100, "Number of lines to show")
    cmd.Flags().StringVar(&logsSince, "since", "", "Show logs since duration (e.g., 5m, 1h)")
    cmd.Flags().BoolVar(&logsTimestamps, "timestamps", true, "Show timestamps")
    cmd.Flags().BoolVar(&logsNoColor, "no-color", false, "Disable colored output")
    cmd.Flags().StringVar(&logsLevel, "level", "all", "Filter by level (info,warn,error,debug,all)")
    cmd.Flags().StringVar(&logsFormat, "format", "text", "Output format (text, json)")
    cmd.Flags().StringVar(&logsOutput, "output", "", "Write to file")
    
    return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
    // Implementation
}
```

## Dashboard Integration

### 1. Update Dashboard Data Model

Add logs endpoint to backend (`internal/dashboard/server.go`):

```go
// Add to setupRoutes()
s.mux.HandleFunc("/api/logs", s.handleGetLogs)
s.mux.HandleFunc("/api/logs/stream", s.handleLogStream)

// handleGetLogs returns recent logs
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
    serviceName := r.URL.Query().Get("service")
    tailStr := r.URL.Query().Get("tail")
    
    logManager := service.GetLogManager(s.projectDir)
    
    var logs []service.LogEntry
    if serviceName != "" {
        buffer, exists := logManager.GetBuffer(serviceName)
        if !exists {
            http.Error(w, "Service not found", http.StatusNotFound)
            return
        }
        logs = buffer.GetRecent(tail)
    } else {
        logs = logManager.GetAllLogs(tail)
    }
    
    json.NewEncoder(w).Encode(logs)
}

// handleLogStream streams logs via WebSocket
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    serviceName := r.URL.Query().Get("service")
    logManager := service.GetLogManager(s.projectDir)
    
    // Subscribe to logs
    var subscriptions []<-chan service.LogEntry
    
    if serviceName != "" {
        buffer, exists := logManager.GetBuffer(serviceName)
        if exists {
            subscriptions = append(subscriptions, buffer.Subscribe())
        }
    } else {
        // Subscribe to all services
        for _, buffer := range logManager.GetAllBuffers() {
            subscriptions = append(subscriptions, buffer.Subscribe())
        }
    }
    
    // Stream logs to WebSocket
    for _, ch := range subscriptions {
        go func(ch <-chan service.LogEntry) {
            for entry := range ch {
                conn.WriteJSON(entry)
            }
        }(ch)
    }
    
    // Keep connection alive
    <-s.stopChan
}
```

### 2. Dashboard UI Updates

#### Add Tabs to Dashboard (`dashboard/src/App.tsx`)

```typescript
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

function App() {
  const [activeTab, setActiveTab] = useState<'services' | 'logs'>('services')
  
  return (
    <div className="min-h-screen bg-background">
      <header>...</header>
      
      <main className="container mx-auto px-4 py-8">
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as any)}>
          <TabsList className="grid w-full grid-cols-2 max-w-md">
            <TabsTrigger value="services">Services</TabsTrigger>
            <TabsTrigger value="logs">Logs</TabsTrigger>
          </TabsList>
          
          <TabsContent value="services">
            {/* Existing service cards */}
          </TabsContent>
          
          <TabsContent value="logs">
            <LogsView />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  )
}
```

#### Create LogsView Component (`dashboard/src/components/LogsView.tsx`)

```typescript
import { useState, useEffect, useRef } from 'react'
import { Select } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Search, Download, Trash2, Pause, Play } from 'lucide-react'

interface LogEntry {
  service: string
  message: string
  level: string
  timestamp: string
  isStderr: boolean
}

export function LogsView() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [selectedService, setSelectedService] = useState<string>('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [autoScroll, setAutoScroll] = useState(true)
  const [isPaused, setIsPaused] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  
  useEffect(() => {
    // Fetch initial logs
    fetchLogs()
    
    // Setup WebSocket for live streaming
    setupWebSocket()
    
    return () => {
      wsRef.current?.close()
    }
  }, [selectedService])
  
  useEffect(() => {
    if (autoScroll && !isPaused) {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, autoScroll, isPaused])
  
  const fetchLogs = async () => {
    const url = selectedService === 'all' 
      ? '/api/logs?tail=500'
      : `/api/logs?service=${selectedService}&tail=500`
    
    const res = await fetch(url)
    const data = await res.json()
    setLogs(data)
  }
  
  const setupWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = selectedService === 'all'
      ? `${protocol}//${window.location.host}/api/logs/stream`
      : `${protocol}//${window.location.host}/api/logs/stream?service=${selectedService}`
    
    const ws = new WebSocket(url)
    
    ws.onmessage = (event) => {
      if (!isPaused) {
        const entry = JSON.parse(event.data)
        setLogs(prev => [...prev, entry].slice(-1000)) // Keep last 1000
      }
    }
    
    wsRef.current = ws
  }
  
  const filteredLogs = logs.filter(log =>
    log.message.toLowerCase().includes(searchTerm.toLowerCase())
  )
  
  const exportLogs = () => {
    const content = filteredLogs
      .map(log => `[${log.timestamp}] [${log.service}] ${log.message}`)
      .join('\n')
    
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `logs-${Date.now()}.txt`
    a.click()
  }
  
  return (
    <div className="space-y-4">
      {/* Controls */}
      <div className="flex gap-4 items-center">
        <Select value={selectedService} onValueChange={setSelectedService}>
          <option value="all">All Services</option>
          {/* Populate from services */}
        </Select>
        
        <div className="relative flex-1">
          <Search className="absolute left-3 top-3 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search logs..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-10"
          />
        </div>
        
        <Button
          variant="outline"
          size="icon"
          onClick={() => setIsPaused(!isPaused)}
        >
          {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
        </Button>
        
        <Button variant="outline" size="icon" onClick={exportLogs}>
          <Download className="w-4 h-4" />
        </Button>
        
        <Button variant="outline" size="icon" onClick={() => setLogs([])}>
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>
      
      {/* Log Display */}
      <div className="bg-card border rounded-lg p-4 h-[600px] overflow-y-auto font-mono text-sm">
        {filteredLogs.length === 0 ? (
          <div className="text-center text-muted-foreground py-12">
            No logs to display
          </div>
        ) : (
          <div className="space-y-1">
            {filteredLogs.map((log, idx) => (
              <div
                key={idx}
                className={`${
                  log.isStderr ? 'text-red-400' : 'text-foreground'
                }`}
              >
                <span className="text-muted-foreground">
                  [{new Date(log.timestamp).toLocaleTimeString()}]
                </span>
                {' '}
                <span className="text-blue-400">[{log.service}]</span>
                {' '}
                {log.message}
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>
    </div>
  )
}
```

## File Storage

Log files will be stored in:
```
.azure/logs/
  ├── api.log
  ├── frontend.log
  └── worker.log
```

## Implementation Phases

### Phase 1: Log Collection Infrastructure
- [ ] Create `LogBuffer` type and implementation
- [ ] Create `LogManager` for managing multiple buffers
- [ ] Integrate log collection into `service.StartService()`
- [ ] Add file logging support

### Phase 2: CLI Command
- [ ] Create `logs.go` command file
- [ ] Implement log filtering and formatting
- [ ] Add follow mode (tail -f)
- [ ] Support time-based filtering
- [ ] Add JSON output format

### Phase 3: Dashboard Backend
- [ ] Add `/api/logs` HTTP endpoint
- [ ] Add `/api/logs/stream` WebSocket endpoint
- [ ] Integrate with LogManager

### Phase 4: Dashboard UI
- [ ] Add tabs to dashboard (Services, Logs)
- [ ] Create LogsView component
- [ ] Implement live streaming via WebSocket
- [ ] Add filtering, search, and export features

## Testing

### Unit Tests
- `logbuffer_test.go`: Test circular buffer, subscription, filtering
- `logmanager_test.go`: Test multi-service management
- `logs_test.go`: Test CLI command logic

### Integration Tests
- `logs_integration_test.go`: Test end-to-end log collection
- Test log streaming to CLI
- Test log streaming to dashboard
- Test file logging

## Security Considerations

1. **Path validation**: Validate service names to prevent path traversal in log files
2. **Size limits**: Enforce maximum buffer size to prevent memory exhaustion
3. **Rate limiting**: Limit WebSocket message rate to prevent DoS
4. **Access control**: Only allow access to logs from the current project

## Performance Considerations

1. **Circular buffer**: Use fixed-size buffer to prevent unbounded memory growth
2. **Efficient broadcasting**: Use channels for pub/sub pattern
3. **File rotation**: Implement log rotation when files exceed size limit
4. **Lazy loading**: Only load logs on demand, not all at startup

## Future Enhancements

- **Log aggregation**: Combine logs from multiple projects
- **Log parsing**: Detect structured logs (JSON) and parse automatically
- **Log filtering**: Advanced filtering by regex, time range, level
- **Log retention**: Configurable retention policies
- **Export formats**: Support CSV, JSON, etc.
- **Cloud sync**: Option to sync logs to Azure Monitor/Application Insights
