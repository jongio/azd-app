import * as React from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface ErrorBoundaryProps {
  children: React.ReactNode
  fallback?: React.ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    // Log error for debugging
    console.error('ErrorBoundary caught an error:', error)
    console.error('Component stack:', errorInfo.componentStack)
  }

  handleReset = (): void => {
    this.setState({ hasError: false, error: undefined })
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback
      }

      // Default fallback UI
      // Show error details in development mode (NODE_ENV !== 'production')
      const isDevelopment = process.env.NODE_ENV !== 'production'

      return (
        <div className="flex flex-col items-center justify-center min-h-[200px] p-6 bg-destructive/10 border border-destructive/30 rounded-lg">
          <div className="flex items-center gap-3 mb-4">
            <AlertTriangle className="w-8 h-8 text-destructive" />
            <h2 className="text-xl font-semibold text-destructive">Something went wrong</h2>
          </div>
          
          <p className="text-sm text-muted-foreground mb-4 text-center max-w-md">
            An unexpected error occurred. Please try again or refresh the page.
          </p>

          {isDevelopment && this.state.error && (
            <div className="w-full max-w-lg mb-4 p-4 bg-muted rounded-md border border-border overflow-auto">
              <p className="text-xs font-mono text-destructive mb-2">
                {this.state.error.name}: {this.state.error.message}
              </p>
              {this.state.error.stack && (
                <pre className="text-xs font-mono text-muted-foreground whitespace-pre-wrap wrap-break-word">
                  {this.state.error.stack}
                </pre>
              )}
            </div>
          )}

          <Button
            variant="outline"
            onClick={this.handleReset}
            className="flex items-center gap-2"
          >
            <RefreshCw className="w-4 h-4" />
            Try Again
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
