import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './styles/editor-design-system.css'
import App from './App.tsx'
import { PreferencesProvider } from './contexts/PreferencesContext'
import { ServiceOperationsProvider } from './contexts/ServiceOperationsContext'
import { ServicesProvider } from './contexts/ServicesContext'
import YamlEditor from './components/editor/YamlEditor'
import { setupEditorApiMocks } from './mocks/editorApiMocks'
import { PreviewPaneTestPage, PreviewToggleTestPage, SchemaFormTestPage } from './test-pages'

const isEditorRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/editor')
const isSchemaFormRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/test/schema-form')
const isPreviewPaneRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/test/preview-pane')
const isPreviewToggleRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/test/preview-toggle')

// In dev/test, mock editor APIs so the YAML editor renders with realistic data during E2E runs.
if (import.meta.env.DEV || import.meta.env.MODE === 'test') {
  setupEditorApiMocks()
}

const rootElement = document.getElementById('root')!

let app: JSX.Element

if (isEditorRoute) {
  app = (
    <StrictMode>
      <YamlEditor />
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
