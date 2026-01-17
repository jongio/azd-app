import { StrictMode, Suspense, lazy } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { PreferencesProvider } from './contexts/PreferencesContext'
import { ServiceOperationsProvider } from './contexts/ServiceOperationsContext'
import { ServicesProvider } from './contexts/ServicesContext'
import { setupEditorApiMocks } from './mocks/editorApiMocks'
import { PreviewPaneTestPage, PreviewToggleTestPage, SchemaFormTestPage } from './test-pages'

// Lazy load the YAML editor to reduce initial bundle size
// Also import editor CSS only when needed
const YamlEditor = lazy(async () => {
  await import('./styles/editor-design-system.css')
  return import('./components/editor/YamlEditor')
})

const isEditorRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/editor')
const isSchemaFormRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/test/schema-form')
const isPreviewPaneRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/test/preview-pane')
const isPreviewToggleRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/test/preview-toggle')

// In dev/test, mock editor APIs so the YAML editor renders with realistic data during E2E runs.
if (import.meta.env.DEV || import.meta.env.MODE === 'test') {
  setupEditorApiMocks()
}

const rootElement = document.getElementById('root')!

// Loading component for editor
const EditorLoading = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="text-center">
      <div className="w-8 h-8 border-4 border-cyan-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
      <p className="text-sm text-slate-600 dark:text-slate-400">Loading Azure YAML Editor...</p>
    </div>
  </div>
)

let app: JSX.Element

if (isEditorRoute) {
  app = (
    <StrictMode>
      <Suspense fallback={<EditorLoading />}>
        <YamlEditor />
      </Suspense>
    </StrictMode>
  )
} else if (isSchemaFormRoute) {
  app = (
    <StrictMode>
      <SchemaFormTestPage />
    </StrictMode>
  )
} else if (isPreviewPaneRoute) {
  app = (
    <StrictMode>
      <PreviewPaneTestPage />
    </StrictMode>
  )
} else if (isPreviewToggleRoute) {
  app = (
    <StrictMode>
      <PreviewToggleTestPage />
    </StrictMode>
  )
} else {
  app = (
    <StrictMode>
      <ServicesProvider>
        <PreferencesProvider>
          <ServiceOperationsProvider>
            <App />
          </ServiceOperationsProvider>
        </PreferencesProvider>
      </ServicesProvider>
    </StrictMode>
  )
}

createRoot(rootElement).render(app)
