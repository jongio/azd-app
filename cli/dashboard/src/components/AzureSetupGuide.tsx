/**
 * AzureSetupGuide - Interactive wizard for setting up Azure logs integration
 * Guides users through workspace config, auth, diagnostic settings, and verification
 */
import * as React from 'react'
import { X, CheckCircle, Circle, ChevronRight, ChevronLeft } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { WorkspaceSetupStep } from './WorkspaceSetupStep'
import { AuthSetupStep } from './AuthSetupStep'
import { DiagnosticSettingsStep } from './DiagnosticSettingsStep'
import { SetupVerification } from './SetupVerification'

// =============================================================================
// Types
// =============================================================================

/**
 * Valid setup step identifiers for the Azure logs configuration wizard.
 * Steps must be completed sequentially: workspace → auth → diagnostic-settings → verification
 */
export type SetupStep = 'workspace' | 'auth' | 'diagnostic-settings' | 'verification'

/**
 * Props for the AzureSetupGuide component.
 * 
 * @property isOpen - Controls visibility of the setup guide modal
 * @property onClose - Callback when user closes the guide
 * @property onComplete - Optional callback when setup is complete and user clicks "Complete Setup"
 * @property initialStep - Optional starting step (useful for deep linking from errors)
 */
export interface AzureSetupGuideProps {
  isOpen: boolean
  onClose: () => void
  onComplete?: () => void
  initialStep?: SetupStep
}

/**
 * Configuration for each step in the setup wizard.
 * Defines the step identifier, display label, and description shown in the stepper.
 */
interface StepConfig {
  id: SetupStep
  label: string
  description: string
}

/**
 * Persisted setup progress data stored in localStorage.
 * Allows users to resume setup where they left off.
 * Expires after PROGRESS_EXPIRY_HOURS to prevent stale state.
 */
interface SetupProgress {
  currentStep: SetupStep
  completedSteps: SetupStep[]
  workspaceId?: string
  timestamp: string
}

// =============================================================================
// Constants
// =============================================================================

const STEPS: StepConfig[] = [
  {
    id: 'workspace',
    label: 'Workspace',
    description: 'Configure Log Analytics workspace',
  },
  {
    id: 'auth',
    label: 'Authentication',
    description: 'Verify authentication and permissions',
  },
  {
    id: 'diagnostic-settings',
    label: 'Diagnostic Settings',
    description: 'Enable logging for services',
  },
  {
    id: 'verification',
    label: 'Verification',
    description: 'Test log connectivity',
  },
]

const PROGRESS_KEY = 'azd-setup-progress'
const PROGRESS_EXPIRY_HOURS = 24

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Load setup progress from localStorage.
 * Returns null if no progress exists, storage is unavailable, or progress has expired.
 * Progress expires after PROGRESS_EXPIRY_HOURS to prevent stale state issues.
 */
function loadProgress(): SetupProgress | null {
  try {
    const stored = localStorage.getItem(PROGRESS_KEY)
    if (!stored) return null

    const progress = JSON.parse(stored) as SetupProgress
    const age = Date.now() - new Date(progress.timestamp).getTime()
    const maxAge = PROGRESS_EXPIRY_HOURS * 60 * 60 * 1000

    // Expire stale progress
    if (age > maxAge) {
      localStorage.removeItem(PROGRESS_KEY)
      return null
    }

    return progress
  } catch {
    return null
  }
}

/**
 * Save setup progress to localStorage for persistence across page reloads.
 * Fails silently if localStorage is unavailable (e.g., private browsing mode).
 */
function saveProgress(progress: SetupProgress): void {
  try {
    localStorage.setItem(PROGRESS_KEY, JSON.stringify(progress))
  } catch {
    // Fail silently if localStorage is unavailable
  }
}

/**
 * Clear saved setup progress from localStorage.
 * Called when setup is completed or user wants to reset progress.
 */
function clearProgress(): void {
  try {
    localStorage.removeItem(PROGRESS_KEY)
  } catch {
    // Fail silently
  }
}

/**
 * Get the zero-based index of a step in the STEPS array.
 * Used for step navigation and validation (can't skip steps).
 */
function getStepIndex(step: SetupStep): number {
  return STEPS.findIndex(s => s.id === step)
}

// =============================================================================
// Stepper Component
// =============================================================================

/**
 * Props for the Stepper component that shows step progress.
 */
interface StepperProps {
  currentStep: SetupStep
  completedSteps: SetupStep[]
  onStepClick?: (step: SetupStep) => void
  isCurrentStepValid?: boolean
}

function Stepper({ currentStep, completedSteps, onStepClick, isCurrentStepValid = false }: Readonly<StepperProps>) {
  return (
    <div className="w-full">
      <div className="flex items-center justify-between">
        {STEPS.map((step, index) => {
          const isCompleted = completedSteps.includes(step.id)
          const isCurrent = step.id === currentStep
          const isClickable = !!onStepClick

          return (
            <React.Fragment key={step.id}>
              {/* Step */}
              <div className="flex flex-col items-center gap-2 flex-1">
                <button
                  type="button"
                  onClick={() => isClickable && onStepClick(step.id)}
                  disabled={!isClickable}
                  className={cn(
                    'w-10 h-10 rounded-full flex items-center justify-center transition-all',
                    'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                    'dark:focus:ring-offset-slate-900',
                    isCompleted && 'bg-emerald-500 text-white shadow-sm',
                    isCurrent && !isCompleted && isCurrentStepValid && 'bg-emerald-500 text-white shadow-sm',
                    isCurrent && !isCompleted && !isCurrentStepValid && 'bg-cyan-500 text-white shadow-sm',
                    !isCurrent && !isCompleted && 'bg-slate-200 dark:bg-slate-700 text-slate-400 dark:text-slate-500',
                    isClickable && 'cursor-pointer hover:scale-105',
                    !isClickable && 'cursor-not-allowed',
                  )}
                  aria-label={`${step.label} - ${isCompleted ? 'Completed' : isCurrent ? 'Current' : 'Upcoming'}`}
                  aria-current={isCurrent ? 'step' : undefined}
                >
                  {isCompleted ? (
                    <CheckCircle className="w-5 h-5" />
                  ) : (
                    <Circle className="w-5 h-5" />
                  )}
                </button>
                <div className="text-center max-w-30">
                  <div
                    className={cn(
                      'text-xs font-medium mb-0.5 transition-colors',
                      (isCurrent || isCompleted) && 'text-slate-900 dark:text-slate-100',
                      !isCurrent && !isCompleted && 'text-slate-500 dark:text-slate-400',
                    )}
                  >
                    {step.label}
                  </div>
                  <div className="text-xs text-slate-500 dark:text-slate-400 hidden sm:block">
                    {step.description}
                  </div>
                </div>
              </div>

              {/* Connector */}
              {index < STEPS.length - 1 && (
                <div className="shrink-0 w-12 sm:w-16 mb-14 sm:mb-16">
                  <div
                    className={cn(
                      'h-0.5 transition-colors',
                      isCompleted ? 'bg-emerald-500' : 'bg-slate-200 dark:bg-slate-700',
                    )}
                  />
                </div>
              )}
            </React.Fragment>
          )
        })}
      </div>
    </div>
  )
}

// =============================================================================
// AzureSetupGuide Component
// =============================================================================

export function AzureSetupGuide({
  isOpen,
  onClose,
  onComplete,
  initialStep,
}: Readonly<AzureSetupGuideProps>) {
  const dialogRef = React.useRef<HTMLDialogElement>(null)

  // Load initial state from localStorage or props
  const [currentStep, setCurrentStep] = React.useState<SetupStep>(() => {
    if (initialStep) return initialStep
    const saved = loadProgress()
    return saved?.currentStep ?? 'workspace'
  })

  const [completedSteps, setCompletedSteps] = React.useState<SetupStep[]>(() => {
    if (initialStep) return [] // Reset completion if deep linking
    const saved = loadProgress()
    return saved?.completedSteps ?? []
  })

  const [isCurrentStepValid, setIsCurrentStepValid] = React.useState(false)

  const currentStepIndex = getStepIndex(currentStep)
  const isFirstStep = currentStepIndex === 0
  const isLastStep = currentStepIndex === STEPS.length - 1

  useEscapeKey(onClose, isOpen)

  // Save progress to localStorage whenever it changes
  React.useEffect(() => {
    if (!isOpen) return

    const progress: SetupProgress = {
      currentStep,
      completedSteps,
      timestamp: new Date().toISOString(),
    }

    saveProgress(progress)
  }, [currentStep, completedSteps, isOpen])

  // Focus management
  React.useEffect(() => {
    if (isOpen && dialogRef.current) {
      const closeButton = dialogRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Navigation handlers
  const handleNext = () => {
    if (isLastStep) {
      // Complete setup
      clearProgress()
      onComplete?.()
      onClose()
      return
    }

    // Mark current step as completed and move to next
    if (!completedSteps.includes(currentStep)) {
      setCompletedSteps(prev => [...prev, currentStep])
    }

    const nextStep = STEPS[currentStepIndex + 1]
    if (nextStep) {
      setCurrentStep(nextStep.id)
    }
  }

  const handleBack = () => {
    if (isFirstStep) return

    const prevStep = STEPS[currentStepIndex - 1]
    if (prevStep) {
      setCurrentStep(prevStep.id)
    }
  }

  const handleSkip = () => {
    if (isLastStep) {
      onClose()
      return
    }

    const nextStep = STEPS[currentStepIndex + 1]
    if (nextStep) {
      setCurrentStep(nextStep.id)
    }
  }

  const handleStepClick = (step: SetupStep) => {
    setCurrentStep(step)
  }

  const handleClose = () => {
    onClose()
  }

  // Render current step content
  const renderStepContent = () => {
    switch (currentStep) {
      case 'workspace':
        return <WorkspaceSetupStep onValidationChange={setIsCurrentStepValid} />
      case 'auth':
        return <AuthSetupStep onValidationChange={setIsCurrentStepValid} />
      case 'diagnostic-settings':
        return <DiagnosticSettingsStep onValidationChange={setIsCurrentStepValid} />
      case 'verification':
        return (
          <SetupVerification 
            onValidationChange={setIsCurrentStepValid} 
            onComplete={onComplete}
            onNavigateToStep={(step) => setCurrentStep(step as SetupStep)}
          />
        )
      default:
        return null
    }
  }

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/50 dark:bg-black/70 animate-fade-in"
        onClick={handleClose}
        aria-hidden="true"
      />

      {/* Dialog */}
      <dialog
        ref={dialogRef}
        open
        aria-labelledby="setup-guide-title"
        className={cn(
          'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
          'w-full max-w-4xl',
          'bg-white dark:bg-slate-900',
          'border border-slate-200 dark:border-slate-700',
          'rounded-2xl shadow-2xl',
          'flex flex-col',
          'max-h-[90vh]',
          'animate-scale-in',
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div>
            <h2
              id="setup-guide-title"
              className="text-lg font-semibold text-slate-900 dark:text-slate-100"
            >
              Azure Logs Setup Guide
            </h2>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-0.5">
              Configure your project to stream logs from Azure services
            </p>
          </div>
          <button
            type="button"
            data-close-button
            onClick={handleClose}
            className="p-2 -mr-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close setup guide"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Stepper */}
        <div className="px-6 py-6 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 shrink-0">
          <Stepper
            currentStep={currentStep}
            completedSteps={completedSteps}
            onStepClick={handleStepClick}
            isCurrentStepValid={isCurrentStepValid}
          />
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {renderStepContent()}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 shrink-0">
          <div className="flex items-center justify-between gap-3">
            {/* Back button */}
            <button
              type="button"
              onClick={handleBack}
              disabled={isFirstStep}
              className={cn(
                'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold border shadow-sm',
                'text-slate-800 dark:text-slate-100 border-slate-200 dark:border-slate-700',
                'bg-white hover:bg-slate-100 dark:bg-slate-900 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                'disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-white dark:disabled:hover:bg-slate-900',
                'transition-colors duration-150',
              )}
            >
              <ChevronLeft className="w-4 h-4" />
              Back
            </button>

            {/* Right side buttons */}
            <div className="flex items-center gap-3">
              {/* Skip button */}
              <button
                type="button"
                onClick={handleSkip}
                className={cn(
                  'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold',
                  'text-slate-700 dark:text-slate-300',
                  'hover:bg-slate-200/70 dark:hover:bg-slate-800',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                  'transition-colors duration-150',
                )}
              >
                {isLastStep ? 'Close' : 'Skip'}
              </button>

              {/* Next/Complete button */}
              <button
                type="button"
                onClick={handleNext}
                disabled={!isCurrentStepValid && !isLastStep}
                className={cn(
                  'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold shadow-sm',
                  'bg-cyan-600 text-white hover:bg-cyan-700',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                  'disabled:opacity-60 disabled:cursor-not-allowed disabled:hover:bg-cyan-600',
                  'transition-colors duration-150',
                )}
              >
                {isLastStep ? 'Complete' : 'Next'}
                {!isLastStep && <ChevronRight className="w-4 h-4" />}
              </button>
            </div>
          </div>
        </div>
      </dialog>
    </>
  )
}

export default AzureSetupGuide
