package executor

// NewHook creates a new Hook with the specified configuration.
// This is a helper to avoid circular imports when other packages need to pass
// hook configurations to the executor package.
//
// Example usage from commands package:
//
//	func convertHook(h *service.Hook) *executor.Hook {
//	    if h == nil {
//	        return nil
//	    }
//	    return executor.NewHook(h.Run, h.Shell, h.ContinueOnError, h.Interactive,
//	        convertPlatformHook(h.Windows), convertPlatformHook(h.Posix))
//	}
//
// Note: This file provides the building blocks. Actual conversion functions
// should live in the package that imports both service and executor (e.g., commands).
func NewHook(run, shell string, continueOnError, interactive bool, windows, posix *PlatformHook) *Hook {
	return &Hook{
		Run:             run,
		Shell:           shell,
		ContinueOnError: continueOnError,
		Interactive:     interactive,
		Windows:         windows,
		Posix:           posix,
	}
}

// NewPlatformHook creates a new PlatformHook with the given configuration.
func NewPlatformHook(run, shell string, continueOnError, interactive *bool) *PlatformHook {
	return &PlatformHook{
		Run:             run,
		Shell:           shell,
		ContinueOnError: continueOnError,
		Interactive:     interactive,
	}
}
