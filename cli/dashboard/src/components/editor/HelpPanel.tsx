/**
 * Help Panel Component - Context-sensitive help sidebar/modal
 * 
 * Provides contextual help based on the current section being edited.
 * Shows descriptions, examples, and links to documentation.
 */

import * as React from 'react'
import { X, BookOpen, ExternalLink, Code2, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface HelpSection {
  /** Section title */
  title: string
  /** Section description */
  description: string
  /** Code examples */
  examples?: Array<{
    title: string
    description?: string
    code: string
    language?: string
  }>
  /** Related links */
  links?: Array<{
    title: string
    url: string
  }>
  /** Common issues and solutions */
  troubleshooting?: Array<{
    issue: string
    solution: string
  }>
}

export interface HelpPanelProps {
  /** Whether the panel is open */
  isOpen: boolean
  /** Callback to close the panel */
  onClose: () => void
  /** Current section being edited (e.g., 'services', 'hooks', 'resources') */
  section?: string
  /** Display mode: 'sidebar' or 'modal' */
  mode?: 'sidebar' | 'modal'
}

/**
 * Help content organized by section
 */
const HELP_CONTENT: Record<string, HelpSection> = {
  name: {
    title: 'Application Name',
    description: 'The application name identifies your project. It must be lowercase, contain only letters, numbers, and hyphens, and start/end with a letter or number.',
    examples: [
      {
        title: 'Valid names',
        code: `name: my-app
name: web-app-2024
name: api-service`,
        language: 'yaml',
      },
    ],
    troubleshooting: [
      {
        issue: 'Name validation error',
        solution: 'Ensure the name is lowercase, uses only letters, numbers, and hyphens, and starts/ends with a letter or number.',
      },
    ],
  },
  services: {
    title: 'Services',
    description: 'Services represent the individual components of your application. Each service can be a web app, API, worker, or container.',
    examples: [
      {
        title: 'Basic web service',
        description: 'A Node.js web application',
        code: `services:
  web:
    project: ./src/web
    host: containerapp
    language: js
    ports:
      - "3000"
    environment:
      NODE_ENV: production`,
        language: 'yaml',
      },
      {
        title: 'API service with database',
        description: 'A Python API that uses a PostgreSQL database',
        code: `services:
  api:
    project: ./src/api
    host: containerapp
    language: python
    ports:
      - "8080"
    uses:
      - db
    environment:
      DATABASE_URL: \${db.connectionString}`,
        language: 'yaml',
      },
    ],
    links: [
      {
        title: 'Service Configuration Reference',
        url: 'https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-schema',
      },
    ],
    troubleshooting: [
      {
        issue: 'Service fails to start',
        solution: 'Check that the project path is correct and all required environment variables are set.',
      },
      {
        issue: 'Port already in use',
        solution: 'Change the port mapping or stop the conflicting service.',
      },
    ],
  },
  resources: {
    title: 'Resources',
    description: 'Resources represent Azure infrastructure components like databases, storage accounts, and key vaults that your services depend on.',
    examples: [
      {
        title: 'PostgreSQL database',
        code: `resources:
  db:
    type: db.postgres
    uses: []`,
        language: 'yaml',
      },
      {
        title: 'Storage account',
        code: `resources:
  storage:
    type: storage`,
        language: 'yaml',
      },
      {
        title: 'Key Vault',
        code: `resources:
  secrets:
    type: keyvault`,
        language: 'yaml',
      },
    ],
    links: [
      {
        title: 'Resource Types Reference',
        url: 'https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-schema#resources',
      },
    ],
  },
  hooks: {
    title: 'Hooks',
    description: 'Hooks allow you to run custom scripts before or after specific azd commands. Use hooks for tasks like database migrations, seed data, or custom deployment steps.',
    examples: [
      {
        title: 'Post-provision hook',
        description: 'Run database migrations after infrastructure is provisioned',
        code: `hooks:
  postprovision:
    run: ./scripts/migrate.sh
    shell: bash`,
        language: 'yaml',
      },
      {
        title: 'Pre-deploy hook',
        description: 'Build assets before deployment',
        code: `hooks:
  predeploy:
    run: npm run build
    shell: sh
    interactive: false`,
        language: 'yaml',
      },
      {
        title: 'Platform-specific hooks',
        description: 'Different commands for Windows and POSIX systems',
        code: `hooks:
  postprovision:
    windows:
      run: .\\scripts\\setup.ps1
      shell: pwsh
    posix:
      run: ./scripts/setup.sh
      shell: bash`,
        language: 'yaml',
      },
    ],
    links: [
      {
        title: 'Hooks Documentation',
        url: 'https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-schema#hooks',
      },
    ],
    troubleshooting: [
      {
        issue: 'Hook script not found',
        solution: 'Ensure the script path is relative to the project root and the file exists.',
      },
      {
        issue: 'Hook fails with permission error',
        solution: 'Make sure the script has execute permissions (chmod +x script.sh on POSIX).',
      },
    ],
  },
  ports: {
    title: 'Port Configuration',
    description: 'Port mappings define how your service exposes network ports. Format: "host:container" or just "port" for the same port on both sides.',
    examples: [
      {
        title: 'Port mapping examples',
        code: `# Map container port 8080 to host port 3000
ports:
  - "3000:8080"

# Use same port on both sides
ports:
  - "8080"

# Bind to specific interface
ports:
  - "127.0.0.1:3000:8080"

# UDP port
ports:
  - "8080/udp"`,
        language: 'yaml',
      },
    ],
  },
  environment: {
    title: 'Environment Variables',
    description: 'Environment variables configure your service at runtime. You can set plain values or reference secrets.',
    examples: [
      {
        title: 'Environment variables',
        code: `environment:
  NODE_ENV: production
  API_URL: https://api.example.com
  DATABASE_URL: \${db.connectionString}`,
        language: 'yaml',
      },
      {
        title: 'Array format with secrets',
        code: `environment:
  - name: NODE_ENV
    value: production
  - name: API_KEY
    secret: \${API_KEY_SECRET}`,
        language: 'yaml',
      },
    ],
  },
  healthcheck: {
    title: 'Health Checks',
    description: 'Health checks monitor your service status. Configure HTTP endpoints, TCP ports, process checks, or output pattern matching.',
    examples: [
      {
        title: 'HTTP health check',
        code: `healthcheck:
  type: http
  path: /health
  interval: 30s
  timeout: 10s
  retries: 3`,
        language: 'yaml',
      },
      {
        title: 'TCP port check',
        code: `healthcheck:
  type: tcp
  interval: 30s`,
        language: 'yaml',
      },
      {
        title: 'Output pattern matching',
        description: 'For watch mode services like TypeScript compiler',
        code: `healthcheck:
  type: output
  pattern: "Found 0 errors"
  timeout: 60s`,
        language: 'yaml',
      },
      {
        title: 'Disable health checks',
        description: 'For build tasks that exit after completion',
        code: `healthcheck:
  disable: true`,
        language: 'yaml',
      },
    ],
  },
  test: {
    title: 'Test Configuration',
    description: 'Configure unit tests, integration tests, and end-to-end tests for your services.',
    examples: [
      {
        title: 'Service-level test configuration',
        code: `test:
  unit:
    command: npm test
    path: tests/unit
    timeout: 5m
  integration:
    command: npm run test:integration
    path: tests/integration
  coverage:
    enabled: true
    threshold: 80`,
        language: 'yaml',
      },
    ],
  },
  logs: {
    title: 'Logging Configuration',
    description: 'Configure log filtering and Azure Log Analytics integration for better log management.',
    examples: [
      {
        title: 'Log filters',
        code: `logs:
  filters:
    exclude:
      - "npm warn"
      - "Debugger listening"
  classifications:
    - text: "ERROR"
      level: error
    - text: "WARN"
      level: warning`,
        language: 'yaml',
      },
      {
        title: 'Azure Log Analytics',
        code: `logs:
  analytics:
    workspace: /subscriptions/.../workspaces/my-workspace
    pollingInterval: 10s
    defaultTimespan: 30m`,
        language: 'yaml',
      },
    ],
  },
}

/**
 * Get help content for a section
 */
function getHelpContent(section?: string): HelpSection {
  if (!section) {
    return {
      title: 'Azure YAML Editor Help',
      description: 'Select a field or section to see contextual help and examples.',
    }
  }

  return HELP_CONTENT[section] || {
    title: section.charAt(0).toUpperCase() + section.slice(1),
    description: `Help content for ${section}.`,
  }
}

/**
 * Help Panel Component
 */
export function HelpPanel({ isOpen, onClose, section, mode = 'sidebar' }: HelpPanelProps) {
  const containerRef = React.useRef<HTMLDivElement>(null)
  const content = getHelpContent(section)

  // Focus trap for modal mode
  React.useEffect(() => {
    if (!isOpen || mode !== 'modal' || !containerRef.current) return

    const focusableElements = containerRef.current.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )

    const firstElement = focusableElements[0]
    const lastElement = focusableElements[focusableElements.length - 1]

    firstElement?.focus()

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }

      if (e.key === 'Tab') {
        if (e.shiftKey) {
          if (document.activeElement === firstElement) {
            e.preventDefault()
            lastElement?.focus()
          }
        } else {
          if (document.activeElement === lastElement) {
            e.preventDefault()
            firstElement?.focus()
          }
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, mode, onClose])

  if (!isOpen) return null

  const panelContent = (
    <div
      ref={containerRef}
      className={cn(
        'bg-background border-l border-border flex flex-col',
        mode === 'modal' && 'rounded-lg border shadow-lg'
      )}
      role={mode === 'modal' ? 'dialog' : undefined}
      aria-label={mode === 'modal' ? 'Help panel' : undefined}
      aria-modal={mode === 'modal' ? true : undefined}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/30">
        <div className="flex items-center gap-2">
          <BookOpen className="w-5 h-5 text-primary" />
          <h2 className="text-lg font-semibold">{content.title}</h2>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          aria-label="Close help panel"
        >
          <X className="w-4 h-4" />
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-6">
        {/* Description */}
        <div className="text-sm text-muted-foreground">
          {content.description}
        </div>

        {/* Examples */}
        {content.examples && content.examples.length > 0 && (
          <div className="space-y-4">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <Code2 className="w-4 h-4" />
              Examples
            </h3>
            {content.examples.map((example, index) => (
              <div key={index} className="space-y-2">
                <div className="text-sm font-medium">{example.title}</div>
                {example.description && (
                  <div className="text-xs text-muted-foreground">{example.description}</div>
                )}
                <pre className="bg-muted p-3 rounded-md text-xs overflow-x-auto border border-border">
                  <code>{example.code}</code>
                </pre>
              </div>
            ))}
          </div>
        )}

        {/* Troubleshooting */}
        {content.troubleshooting && content.troubleshooting.length > 0 && (
          <div className="space-y-3">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <AlertCircle className="w-4 h-4" />
              Common Issues
            </h3>
            {content.troubleshooting.map((item, index) => (
              <div key={index} className="space-y-1 p-3 bg-muted/50 rounded-md border border-border">
                <div className="text-sm font-medium">{item.issue}</div>
                <div className="text-xs text-muted-foreground">{item.solution}</div>
              </div>
            ))}
          </div>
        )}

        {/* Links */}
        {content.links && content.links.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-semibold">Learn More</h3>
            {content.links.map((link, index) => (
              <a
                key={index}
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-sm text-primary hover:underline"
              >
                <ExternalLink className="w-3.5 h-3.5" />
                {link.title}
              </a>
            ))}
          </div>
        )}
      </div>
    </div>
  )

  if (mode === 'modal') {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        onClick={onClose}
      >
        <div
          className="w-full max-w-3xl max-h-[90vh] overflow-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          {panelContent}
        </div>
      </div>
    )
  }

  return (
    <div className="w-96 h-full overflow-hidden">
      {panelContent}
    </div>
  )
}
