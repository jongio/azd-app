# AZD App Dashboard

React/TypeScript dashboard for the azd app extension, providing real-time monitoring of local services and Azure deployments.

## Tech Stack

- **React 18** - UI framework
- **TypeScript 5** - Type safety
- **Vite** - Build tool with HMR
- **Tailwind CSS 4** - Styling
- **Vitest** - Testing framework
- **WebSocket** - Real-time updates from backend

## Development

### Quick Start

```bash
# Install dependencies
npm install

# Start dev server with hot reload (uses mock data)
npm run dev
```

The dev server runs at `http://localhost:5173` with hot module replacement (HMR).

### Development Modes

#### 1. Full Stack Development (Recommended)

Use this for testing with the real Go backend and WebSocket updates:

```bash
cd ..  # Back to cli directory
mage watch
```

This single command automatically:
- Watches and rebuilds Go backend on changes
- Watches and rebuilds dashboard on changes
- Reinstalls the extension when either changes

#### 2. Standalone UI Development (Fastest)

Use this when working on UI/UX without needing the real backend:

```bash
npm run dev
```

- Hot module replacement for instant updates
- Mock service data in `src/hooks/useServices.ts`
- No backend required

#### 3. Production Build

Build the production bundle (output goes to `../src/internal/dashboard/dist`):

```bash
npm run build
```

## Project Structure

```
dashboard/
├── src/
│   ├── components/         # React components
│   │   ├── ServiceCard.tsx
│   │   ├── ServiceTable.tsx
│   │   ├── StatusCell.tsx
│   │   ├── LogsView.tsx
│   │   └── ui/            # Reusable UI components
│   ├── hooks/             # Custom React hooks
│   │   └── useServices.ts # Service data + WebSocket
│   ├── lib/               # Utilities
│   ├── test/              # Test utilities
│   ├── App.tsx            # Main app component
│   ├── main.tsx           # Entry point
│   └── types.ts           # TypeScript types
├── index.html             # HTML template
├── vite.config.ts         # Vite configuration
├── vitest.config.ts       # Test configuration
└── package.json
```

## Testing

```bash
# Run tests once
npm test

# Run tests in watch mode
npm run test:ui

# Generate coverage report
npm run test:coverage
```

## API Integration

The dashboard connects to the Go backend via:

- **HTTP API**: `/api/services`, `/api/project`, `/api/logs`
- **WebSocket**: `/api/ws` for real-time service updates

### WebSocket Message Format

The backend sends messages with this structure:

```typescript
{
  type: 'services',
  services: ServiceInfo[]
}
```

Where `ServiceInfo` includes:
- `name`: Service name
- `local`: Local development info (status, health, URL, port, PID)
- `azure`: Azure deployment info (URL, resource name)
- `language`, `framework`: Service metadata

## Building for Production

The production build is embedded in the Go binary via `go:embed`:

```go
//go:embed dist
var staticFiles embed.FS
```

Output location: `../src/internal/dashboard/dist/`

## Environment Variables

None required - configuration is determined by the API endpoints.

## Troubleshooting

### Dashboard not updating in real-time

1. Check WebSocket connection in browser DevTools (Network → WS tab)
2. Verify backend is running with `azd app run`
3. Check for WebSocket message type `'services'` in the console

### Build fails

```bash
# Clean install
rm -rf node_modules package-lock.json
npm install
npm run build
```

### TypeScript errors

```bash
# Check for type errors
npx tsc --noEmit
```

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development workflow and guidelines.
