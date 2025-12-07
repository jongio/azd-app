package testing

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/logging"
)

// FileWatcher monitors files for changes and triggers test re-runs
type FileWatcher struct {
	paths          []string
	ignorePatterns []string
	lastCheck      map[string]time.Time
	pollInterval   time.Duration

	// Debouncing
	debounceDelay  time.Duration
	pendingChanges map[string]time.Time
	pendingMu      sync.Mutex
	debounceTimer  *time.Timer

	// Service tracking
	servicePathMap map[string]string // file path -> service name
	lastRunTime    time.Time

	// Options
	clearConsole    bool
	showElapsedTime bool
}

// WatcherOption configures the file watcher
type WatcherOption func(*FileWatcher)

// WithDebounceDelay sets the debounce delay for file changes
func WithDebounceDelay(delay time.Duration) WatcherOption {
	return func(w *FileWatcher) {
		w.debounceDelay = delay
	}
}

// WithClearConsole enables clearing the console between runs
func WithClearConsole(clear bool) WatcherOption {
	return func(w *FileWatcher) {
		w.clearConsole = clear
	}
}

// WithShowElapsedTime enables showing elapsed time since last run
func WithShowElapsedTime(show bool) WatcherOption {
	return func(w *FileWatcher) {
		w.showElapsedTime = show
	}
}

// WithServicePathMap sets the mapping from file paths to service names
func WithServicePathMap(mapping map[string]string) WatcherOption {
	return func(w *FileWatcher) {
		w.servicePathMap = mapping
	}
}

// NewFileWatcher creates a new file watcher for the given paths
func NewFileWatcher(paths []string, opts ...WatcherOption) *FileWatcher {
	w := &FileWatcher{
		paths:        paths,
		lastCheck:    make(map[string]time.Time),
		pollInterval: DefaultPollInterval,
		ignorePatterns: []string{
			"node_modules",
			".git",
			"__pycache__",
			"*.pyc",
			"bin",
			"obj",
			"dist",
			"build",
			"coverage",
			"test-results",
			".DS_Store",
		},
		debounceDelay:   DefaultDebounceDelay,
		pendingChanges:  make(map[string]time.Time),
		servicePathMap:  make(map[string]string),
		clearConsole:    false,
		showElapsedTime: true,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// WatchCallback is called when changes are detected
// changedServices is the list of services with changes (empty means run all)
type WatchCallback func(changedServices []string) error

// Watch monitors files for changes and calls the callback when changes are detected
func (w *FileWatcher) Watch(ctx context.Context, callback func() error) error {
	return w.WatchWithServiceFilter(ctx, func(services []string) error {
		return callback()
	})
}

// WatchWithServiceFilter monitors files and provides affected services to callback
func (w *FileWatcher) WatchWithServiceFilter(ctx context.Context, callback WatchCallback) error {
	// Set up signal handling for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle signals
	go func() {
		<-sigChan
		log := logging.NewLogger("watch")
		if !logging.IsStructured() {
			fmt.Println("\n\n👋 Received interrupt signal, stopping watcher...")
		}
		log.Info("received interrupt signal", "event", "watch_interrupted")
		cancel()
	}()

	// Initial run
	w.lastRunTime = time.Now()
	if err := callback(nil); err != nil {
		log := logging.NewLogger("watch")
		if !logging.IsStructured() {
			fmt.Printf("Initial test run failed: %v\n", err)
		}
		log.Error("initial test run failed", "event", "test_failed", "error", err.Error())
	}

	// Initialize file modification times
	if err := w.scanFiles(); err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	w.printWatchingMessage()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.cleanup()
			log := logging.NewLogger("watch")
			if !logging.IsStructured() {
				fmt.Println("👋 Stopped watching")
			}
			log.Info("stopped watching", "event", "watch_stopped")
			return nil
		case <-ticker.C:
			changedFiles, err := w.checkForChanges()
			if err != nil {
				log := logging.NewLogger("watch")
				if !logging.IsStructured() {
					fmt.Printf("Error checking for changes: %v\n", err)
				}
				log.Error("error checking for changes", "error", err.Error())
				continue
			}

			if len(changedFiles) > 0 {
				// Add to pending changes for debouncing
				w.pendingMu.Lock()
				for _, file := range changedFiles {
					w.pendingChanges[file] = time.Now()
				}
				w.scheduleDebouncedCallback(ctx, callback)
				w.pendingMu.Unlock()
			}
		}
	}
}

// scheduleDebouncedCallback schedules a callback after debounce delay
func (w *FileWatcher) scheduleDebouncedCallback(ctx context.Context, callback WatchCallback) {
	// Cancel existing timer if any
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	w.debounceTimer = time.AfterFunc(w.debounceDelay, func() {
		w.pendingMu.Lock()
		changes := make(map[string]time.Time)
		for k, v := range w.pendingChanges {
			changes[k] = v
		}
		w.pendingChanges = make(map[string]time.Time)
		w.pendingMu.Unlock()

		if len(changes) == 0 {
			return
		}

		// Determine affected services
		affectedServices := w.getAffectedServices(changes)

		// Clear console if enabled
		if w.clearConsole {
			w.clearScreen()
		}

		// Build list of changed file names for logging
		changedFileNames := make([]string, 0, len(changes))
		for file := range changes {
			changedFileNames = append(changedFileNames, filepath.Base(file))
		}

		// Structured logging for machine-parseable output
		log := logging.NewLogger("watch")
		elapsed := time.Since(w.lastRunTime)
		log.Info("changes detected",
			"event", "file_changed",
			"file_count", len(changes),
			"files", changedFileNames,
			"affected_services", affectedServices,
			"elapsed_sec", elapsed.Seconds(),
		)

		// Console output with emojis for human-friendly display
		if !logging.IsStructured() {
			fmt.Printf("\n🔄 Changes detected in %d file(s)", len(changes))
			if w.showElapsedTime {
				fmt.Printf(" (%.1fs since last run)", elapsed.Seconds())
			}
			fmt.Println()

			// Show affected files
			for file := range changes {
				fmt.Printf("   📝 %s\n", filepath.Base(file))
			}

			// Show affected services if any
			if len(affectedServices) > 0 {
				fmt.Printf("   🎯 Affected services: %s\n", strings.Join(affectedServices, ", "))
			}

			fmt.Println("\n   Re-running tests...")
		}

		w.lastRunTime = time.Now()

		if err := callback(affectedServices); err != nil {
			if !logging.IsStructured() {
				fmt.Printf("Test run failed: %v\n", err)
			}
			log.Error("test run failed", "event", "test_failed", "error", err.Error())
		}

		w.printWatchingMessage()
	})
}

// getAffectedServices determines which services are affected by changed files
func (w *FileWatcher) getAffectedServices(changes map[string]time.Time) []string {
	serviceSet := make(map[string]bool)

	for changedFile := range changes {
		// Check direct mapping
		if service, ok := w.servicePathMap[changedFile]; ok {
			serviceSet[service] = true
			continue
		}

		// Check if file is under any service path
		for _, path := range w.paths {
			absPath, _ := filepath.Abs(path)
			changedAbs, _ := filepath.Abs(changedFile)

			if strings.HasPrefix(changedAbs, absPath) {
				// Find the service name from servicePathMap
				for filePath, service := range w.servicePathMap {
					serviceAbs, _ := filepath.Abs(filePath)
					if strings.HasPrefix(changedAbs, serviceAbs) {
						serviceSet[service] = true
						break
					}
				}
			}
		}
	}

	// Convert map to slice
	services := make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		services = append(services, service)
	}

	return services
}

// printWatchingMessage prints the watching message
func (w *FileWatcher) printWatchingMessage() {
	log := logging.NewLogger("watch")
	log.Info("watching for changes", "event", "watch_started", "paths", w.paths)
	if !logging.IsStructured() {
		fmt.Println("\n👀 Watching for file changes... (Press Ctrl+C to stop)")
	}
}

// clearScreen clears the terminal screen
func (w *FileWatcher) clearScreen() {
	// ANSI escape code to clear screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")
}

// cleanup performs cleanup when stopping the watcher
func (w *FileWatcher) cleanup() {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
}

// scanFiles initializes the file modification times
func (w *FileWatcher) scanFiles() error {
	for _, path := range w.paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories and ignored patterns
			if info.IsDir() || w.shouldIgnore(filePath) {
				if info.IsDir() && w.shouldIgnore(filePath) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only track source files and test files
			if w.isRelevantFile(filePath) {
				w.lastCheck[filePath] = info.ModTime()
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// checkForChanges checks if any files have been modified and returns changed files
func (w *FileWatcher) checkForChanges() ([]string, error) {
	var changedFiles []string

	for _, path := range w.paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories and ignored patterns
			if info.IsDir() || w.shouldIgnore(filePath) {
				if info.IsDir() && w.shouldIgnore(filePath) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only check relevant files
			if !w.isRelevantFile(filePath) {
				return nil
			}

			lastMod, exists := w.lastCheck[filePath]
			if !exists || info.ModTime().After(lastMod) {
				changedFiles = append(changedFiles, filePath)
				w.lastCheck[filePath] = info.ModTime()
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return changedFiles, nil
}

// shouldIgnore checks if a path should be ignored
func (w *FileWatcher) shouldIgnore(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range w.ignorePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		// Also check if the path contains the pattern
		if filepath.Base(filepath.Dir(path)) == pattern {
			return true
		}
	}
	return false
}

// isRelevantFile checks if a file is relevant for test watching
func (w *FileWatcher) isRelevantFile(path string) bool {
	ext := filepath.Ext(path)
	relevantExts := map[string]bool{
		".js":   true,
		".jsx":  true,
		".ts":   true,
		".tsx":  true,
		".mjs":  true,
		".cjs":  true,
		".py":   true,
		".cs":   true,
		".go":   true,
		".java": true,
	}

	return relevantExts[ext]
}

// SetServicePathMap configures the mapping of service paths to names
func (w *FileWatcher) SetServicePathMap(services map[string]string) {
	w.servicePathMap = services
}

// AddIgnorePattern adds a pattern to ignore during file watching
func (w *FileWatcher) AddIgnorePattern(pattern string) {
	w.ignorePatterns = append(w.ignorePatterns, pattern)
}

// SetPollInterval sets the polling interval for file changes
func (w *FileWatcher) SetPollInterval(interval time.Duration) {
	w.pollInterval = interval
}
