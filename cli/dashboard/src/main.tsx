import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { PreferencesProvider } from './contexts/PreferencesContext'
import { ServiceOperationsProvider } from './contexts/ServiceOperationsContext'
import { ServicesProvider } from './contexts/ServicesContext'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ServicesProvider>
      <PreferencesProvider>
        <ServiceOperationsProvider>
          <App />
        </ServiceOperationsProvider>
      </PreferencesProvider>
    </ServicesProvider>
  </StrictMode>,
)
