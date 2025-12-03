import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { DesignModeProvider } from './contexts/DesignModeContext'
import { ServiceOperationsProvider } from './contexts/ServiceOperationsContext'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <DesignModeProvider>
      <ServiceOperationsProvider>
        <App />
      </ServiceOperationsProvider>
    </DesignModeProvider>
  </StrictMode>,
)
