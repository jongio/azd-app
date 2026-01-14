/**
 * Template Tab Component
 * Displays a grid of pre-built azure.yaml templates
 */

import * as React from 'react'
import { cn } from '@/lib/utils'
import { FileText, Check } from 'lucide-react'

interface Template {
  id: string
  name: string
  description: string
  content: string
  category: string
  icon?: string
}

// Built-in templates
const TEMPLATES: Template[] = [
  {
    id: 'minimal',
    name: 'Minimal',
    description: 'Minimal azure.yaml with just the basics',
    category: 'starter',
    icon: '📄',
    content: `name: my-app
services:
  api:
    host: containerapp
    language: node
    project: ./src/api
`,
  },
  {
    id: 'web-app',
    name: 'Web Application',
    description: 'Frontend + backend web application',
    category: 'web',
    icon: '🌐',
    content: `name: my-web-app
services:
  web:
    host: containerapp
    language: node
    project: ./src/web
    ports:
      - "3000:3000"
    environment:
      API_URL: http://localhost:8080
  api:
    host: containerapp
    language: node
    project: ./src/api
    ports:
      - "8080:8080"
    healthcheck:
      test: "curl -f http://localhost:8080/health || exit 1"
      interval: 30s
      timeout: 5s
      retries: 3
`,
  },
  {
    id: 'microservices',
    name: 'Microservices',
    description: 'Multiple services with storage and database',
    category: 'architecture',
    icon: '🔧',
    content: `name: my-microservices
services:
  gateway:
    host: containerapp
    language: node
    project: ./src/gateway
    ports:
      - "8080:8080"
  auth:
    host: containerapp
    language: node
    project: ./src/auth
    ports:
      - "8081:8081"
  data:
    host: containerapp
    language: node
    project: ./src/data
    ports:
      - "8082:8082"
  azurite:
    host: containerapp
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"
      - "10001:10001"
  postgres:
    host: containerapp
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: dev123
      POSTGRES_DB: myapp
`,
  },
  {
    id: 'full-stack',
    name: 'Full Stack',
    description: 'Complete application with frontend, backend, database, cache, and storage',
    category: 'complete',
    icon: '🚀',
    content: `name: my-fullstack-app
services:
  web:
    host: containerapp
    language: react
    project: ./src/web
    ports:
      - "3000:3000"
    environment:
      API_URL: http://localhost:8080
  api:
    host: containerapp
    language: node
    project: ./src/api
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgresql://postgres:dev123@localhost:5432/myapp
      REDIS_URL: redis://localhost:6379
      STORAGE_CONNECTION_STRING: UseDevelopmentStorage=true
    healthcheck:
      test: "curl -f http://localhost:8080/health || exit 1"
      interval: 30s
      timeout: 5s
      retries: 3
  postgres:
    host: containerapp
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: dev123
      POSTGRES_DB: myapp
  redis:
    host: containerapp
    image: redis:7-alpine
    ports:
      - "6379:6379"
  azurite:
    host: containerapp
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"
      - "10001:10001"
      - "10002:10002"
`,
  },
]

export interface TemplateTabProps {
  onSelectTemplate: (yaml: string) => void
}

/**
 * Template Tab Component
 */
export function TemplateTab({ onSelectTemplate }: TemplateTabProps) {
  const [selectedTemplateId, setSelectedTemplateId] = React.useState<string | null>(null)

  const handleSelectTemplate = (template: Template) => {
    setSelectedTemplateId(template.id)
    onSelectTemplate(template.content)
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Select a pre-built template to get started quickly
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {TEMPLATES.map((template) => (
          <button
            key={template.id}
            type="button"
            onClick={() => handleSelectTemplate(template)}
            className={cn(
              'p-4 rounded-lg border-2 text-left transition-all duration-150',
              'hover:shadow-md',
              selectedTemplateId === template.id
                ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
            )}
          >
            <div className="flex items-start justify-between gap-3 mb-2">
              <div className="flex items-center gap-3">
                <div className="text-2xl" aria-hidden="true">
                  {template.icon || <FileText className="w-6 h-6 text-slate-400" />}
                </div>
                <div>
                  <h3 className={cn(
                    'font-semibold text-sm',
                    selectedTemplateId === template.id
                      ? 'text-cyan-900 dark:text-cyan-100'
                      : 'text-slate-900 dark:text-slate-100'
                  )}>
                    {template.name}
                  </h3>
                  <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                    {template.category}
                  </p>
                </div>
              </div>
              {selectedTemplateId === template.id && (
                <Check className="w-5 h-5 text-cyan-600 dark:text-cyan-400 flex-shrink-0" />
              )}
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              {template.description}
            </p>
          </button>
        ))}
      </div>
    </div>
  )
}
