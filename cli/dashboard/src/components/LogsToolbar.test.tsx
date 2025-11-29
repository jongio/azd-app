import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsToolbar, type LogsToolbarProps } from './LogsToolbar'

// Default props for testing
function createDefaultProps(overrides: Partial<LogsToolbarProps> = {}): LogsToolbarProps {
  return {
    viewMode: 'grid',
    onViewModeChange: vi.fn(),
    isFullscreen: false,
    onFullscreenChange: vi.fn(),
    isPaused: false,
    onPauseChange: vi.fn(),
    autoScrollEnabled: true,
    onAutoScrollChange: vi.fn(),
    searchTerm: '',
    onSearchChange: vi.fn(),
    onClearAll: vi.fn(),
    onExportAll: vi.fn(),
    onOpenSettings: vi.fn(),
    onStartAll: vi.fn().mockResolvedValue(undefined),
    onStopAll: vi.fn().mockResolvedValue(undefined),
    onRestartAll: vi.fn().mockResolvedValue(undefined),
    isBulkOperationInProgress: vi.fn().mockReturnValue(false),
    bulkOperation: null,
    gridColumns: 2,
    onGridColumnsChange: vi.fn(),
    ...overrides,
  }
}

describe('LogsToolbar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('rendering', () => {
    it('should render the toolbar with correct ARIA role', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      const toolbar = screen.getByRole('toolbar', { name: 'Log viewer controls' })
      expect(toolbar).toBeInTheDocument()
    })

    it('should render search input with placeholder', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      const searchInput = screen.getByPlaceholderText('Search logs...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should render pause button', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      const pauseButton = screen.getByRole('button', { name: /pause log stream/i })
      expect(pauseButton).toBeInTheDocument()
    })

    it('should render auto-scroll toggle button', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      const scrollButton = screen.getByRole('button', { name: /disable auto-scroll/i })
      expect(scrollButton).toBeInTheDocument()
    })

    it('should render service action buttons', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      expect(screen.getByTitle('Start All (Ctrl+Shift+S)')).toBeInTheDocument()
      expect(screen.getByTitle('Restart All (Ctrl+Shift+R)')).toBeInTheDocument()
      expect(screen.getByTitle('Stop All (Ctrl+Shift+X)')).toBeInTheDocument()
    })

    it('should render log action buttons', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      expect(screen.getByTitle('Clear All Logs')).toBeInTheDocument()
      expect(screen.getByTitle('Export All Logs')).toBeInTheDocument()
    })

    it('should render fullscreen and settings buttons', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      expect(screen.getByTitle('Enter Fullscreen (F11)')).toBeInTheDocument()
      expect(screen.getByTitle('Settings (Ctrl+,)')).toBeInTheDocument()
    })
  })

  describe('view mode toggle', () => {
    it('should show view mode toggle when not in fullscreen', () => {
      const props = createDefaultProps({ isFullscreen: false })
      render(<LogsToolbar {...props} />)

      const gridButton = screen.getByRole('button', { name: /grid/i })
      expect(gridButton).toBeInTheDocument()
    })

    it('should not show view mode toggle in fullscreen mode', () => {
      const props = createDefaultProps({ isFullscreen: true })
      render(<LogsToolbar {...props} />)

      const gridButtons = screen.queryAllByRole('button', { name: /grid/i })
      expect(gridButtons.length).toBe(0)
    })

    it('should highlight active view mode', () => {
      const props = createDefaultProps({ viewMode: 'grid' })
      render(<LogsToolbar {...props} />)

      const gridButton = screen.getByRole('button', { name: /grid/i })
      expect(gridButton).toHaveAttribute('aria-pressed', 'true')
    })

    it('should call onViewModeChange when clicking view mode button', async () => {
      const user = userEvent.setup()
      const onViewModeChange = vi.fn()
      const props = createDefaultProps({ viewMode: 'grid', onViewModeChange })
      render(<LogsToolbar {...props} />)

      const unifiedButton = screen.getByRole('button', { name: /unified/i })
      await user.click(unifiedButton)

      expect(onViewModeChange).toHaveBeenCalledWith('unified')
    })
  })

  describe('grid columns control', () => {
    it('should show grid columns control in grid mode', () => {
      const props = createDefaultProps({ viewMode: 'grid' })
      render(<LogsToolbar {...props} />)

      expect(screen.getByRole('group', { name: /grid columns/i })).toBeInTheDocument()
    })

    it('should not show grid columns control in unified mode', () => {
      const props = createDefaultProps({ viewMode: 'unified' })
      render(<LogsToolbar {...props} />)

      expect(screen.queryByRole('group', { name: /grid columns/i })).not.toBeInTheDocument()
    })

    it('should display current column count', () => {
      const props = createDefaultProps({ gridColumns: 3 })
      render(<LogsToolbar {...props} />)

      expect(screen.getByText('3')).toBeInTheDocument()
    })

    it('should call onGridColumnsChange when increasing columns', async () => {
      const user = userEvent.setup()
      const onGridColumnsChange = vi.fn()
      const props = createDefaultProps({ gridColumns: 2, onGridColumnsChange })
      render(<LogsToolbar {...props} />)

      const increaseButton = screen.getByRole('button', { name: /increase columns/i })
      await user.click(increaseButton)

      expect(onGridColumnsChange).toHaveBeenCalledWith(3)
    })

    it('should call onGridColumnsChange when decreasing columns', async () => {
      const user = userEvent.setup()
      const onGridColumnsChange = vi.fn()
      const props = createDefaultProps({ gridColumns: 3, onGridColumnsChange })
      render(<LogsToolbar {...props} />)

      const decreaseButton = screen.getByRole('button', { name: /decrease columns/i })
      await user.click(decreaseButton)

      expect(onGridColumnsChange).toHaveBeenCalledWith(2)
    })

    it('should disable decrease button when at minimum (1)', () => {
      const props = createDefaultProps({ gridColumns: 1 })
      render(<LogsToolbar {...props} />)

      const decreaseButton = screen.getByRole('button', { name: /decrease columns/i })
      expect(decreaseButton).toBeDisabled()
    })

    it('should disable increase button when at maximum (6)', () => {
      const props = createDefaultProps({ gridColumns: 6 })
      render(<LogsToolbar {...props} />)

      const increaseButton = screen.getByRole('button', { name: /increase columns/i })
      expect(increaseButton).toBeDisabled()
    })

    it('should not show grid columns control in fullscreen mode', () => {
      const props = createDefaultProps({ isFullscreen: true, viewMode: 'grid' })
      render(<LogsToolbar {...props} />)

      expect(screen.queryByRole('group', { name: /grid columns/i })).not.toBeInTheDocument()
    })
  })

  describe('search input', () => {
    it('should display current search term', () => {
      const props = createDefaultProps({ searchTerm: 'test query' })
      render(<LogsToolbar {...props} />)

      const searchInput = screen.getByPlaceholderText('Search logs...')
      expect(searchInput).toHaveValue('test query')
    })

    it('should call onSearchChange when typing', async () => {
      const user = userEvent.setup()
      const onSearchChange = vi.fn()
      const props = createDefaultProps({ onSearchChange })
      render(<LogsToolbar {...props} />)

      const searchInput = screen.getByPlaceholderText('Search logs...')
      await user.type(searchInput, 'error')

      expect(onSearchChange).toHaveBeenCalled()
    })

    it('should have correct aria-label for accessibility', () => {
      const props = createDefaultProps()
      render(<LogsToolbar {...props} />)

      const searchInput = screen.getByLabelText('Search logs')
      expect(searchInput).toBeInTheDocument()
    })
  })

  describe('pause/resume toggle', () => {
    it('should show pause icon when not paused', () => {
      const props = createDefaultProps({ isPaused: false })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /pause log stream/i })
      expect(button).toHaveAttribute('aria-pressed', 'false')
    })

    it('should show resume label when paused', () => {
      const props = createDefaultProps({ isPaused: true })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /resume log stream/i })
      expect(button).toHaveAttribute('aria-pressed', 'true')
    })

    it('should call onPauseChange when clicked', async () => {
      const user = userEvent.setup()
      const onPauseChange = vi.fn()
      const props = createDefaultProps({ isPaused: false, onPauseChange })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /pause log stream/i })
      await user.click(button)

      expect(onPauseChange).toHaveBeenCalledWith(true)
    })

    it('should call onPauseChange with false when resuming', async () => {
      const user = userEvent.setup()
      const onPauseChange = vi.fn()
      const props = createDefaultProps({ isPaused: true, onPauseChange })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /resume log stream/i })
      await user.click(button)

      expect(onPauseChange).toHaveBeenCalledWith(false)
    })
  })

  describe('auto-scroll toggle', () => {
    it('should show enabled state when autoScrollEnabled is true', () => {
      const props = createDefaultProps({ autoScrollEnabled: true })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /disable auto-scroll/i })
      expect(button).toHaveAttribute('aria-pressed', 'true')
    })

    it('should show disabled state when autoScrollEnabled is false', () => {
      const props = createDefaultProps({ autoScrollEnabled: false })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /enable auto-scroll/i })
      expect(button).toHaveAttribute('aria-pressed', 'false')
    })

    it('should call onAutoScrollChange when clicked', async () => {
      const user = userEvent.setup()
      const onAutoScrollChange = vi.fn()
      const props = createDefaultProps({ autoScrollEnabled: true, onAutoScrollChange })
      render(<LogsToolbar {...props} />)

      const button = screen.getByRole('button', { name: /disable auto-scroll/i })
      await user.click(button)

      expect(onAutoScrollChange).toHaveBeenCalledWith(false)
    })
  })

  describe('service actions', () => {
    it('should call onStartAll when Start All is clicked', async () => {
      const user = userEvent.setup()
      const onStartAll = vi.fn().mockResolvedValue(undefined)
      const props = createDefaultProps({ onStartAll })
      render(<LogsToolbar {...props} />)

      const startButton = screen.getByTitle('Start All (Ctrl+Shift+S)')
      await user.click(startButton)

      expect(onStartAll).toHaveBeenCalled()
    })

    it('should call onRestartAll when Restart All is clicked', async () => {
      const user = userEvent.setup()
      const onRestartAll = vi.fn().mockResolvedValue(undefined)
      const props = createDefaultProps({ onRestartAll })
      render(<LogsToolbar {...props} />)

      const restartButton = screen.getByTitle('Restart All (Ctrl+Shift+R)')
      await user.click(restartButton)

      expect(onRestartAll).toHaveBeenCalled()
    })

    it('should call onStopAll when Stop All is clicked', async () => {
      const user = userEvent.setup()
      const onStopAll = vi.fn().mockResolvedValue(undefined)
      const props = createDefaultProps({ onStopAll })
      render(<LogsToolbar {...props} />)

      const stopButton = screen.getByTitle('Stop All (Ctrl+Shift+X)')
      await user.click(stopButton)

      expect(onStopAll).toHaveBeenCalled()
    })

    it('should disable service buttons when bulk operation is in progress', () => {
      const props = createDefaultProps({
        isBulkOperationInProgress: vi.fn().mockReturnValue(true),
      })
      render(<LogsToolbar {...props} />)

      expect(screen.getByTitle('Start All (Ctrl+Shift+S)')).toBeDisabled()
      expect(screen.getByTitle('Restart All (Ctrl+Shift+R)')).toBeDisabled()
      expect(screen.getByTitle('Stop All (Ctrl+Shift+X)')).toBeDisabled()
    })

    it('should show spinner on restart button when restart operation is in progress', () => {
      const props = createDefaultProps({
        isBulkOperationInProgress: vi.fn().mockReturnValue(true),
        bulkOperation: 'restart',
      })
      render(<LogsToolbar {...props} />)

      const restartButton = screen.getByTitle('Restart All (Ctrl+Shift+R)')
      expect(restartButton.querySelector('.animate-spin')).toBeInTheDocument()
    })
  })

  describe('log actions', () => {
    it('should call onClearAll when Clear is clicked', async () => {
      const user = userEvent.setup()
      const onClearAll = vi.fn()
      const props = createDefaultProps({ onClearAll })
      render(<LogsToolbar {...props} />)

      const clearButton = screen.getByTitle('Clear All Logs')
      await user.click(clearButton)

      expect(onClearAll).toHaveBeenCalled()
    })

    it('should call onExportAll when Export is clicked', async () => {
      const user = userEvent.setup()
      const onExportAll = vi.fn()
      const props = createDefaultProps({ onExportAll })
      render(<LogsToolbar {...props} />)

      const exportButton = screen.getByTitle('Export All Logs')
      await user.click(exportButton)

      expect(onExportAll).toHaveBeenCalled()
    })

    it('should not show Export button in fullscreen mode', () => {
      const props = createDefaultProps({ isFullscreen: true })
      render(<LogsToolbar {...props} />)

      expect(screen.queryByTitle('Export All Logs')).not.toBeInTheDocument()
    })
  })

  describe('view actions', () => {
    it('should call onFullscreenChange when fullscreen button is clicked', async () => {
      const user = userEvent.setup()
      const onFullscreenChange = vi.fn()
      const props = createDefaultProps({ isFullscreen: false, onFullscreenChange })
      render(<LogsToolbar {...props} />)

      const fullscreenButton = screen.getByTitle('Enter Fullscreen (F11)')
      await user.click(fullscreenButton)

      expect(onFullscreenChange).toHaveBeenCalledWith(true)
    })

    it('should show Exit Fullscreen when in fullscreen mode', () => {
      const props = createDefaultProps({ isFullscreen: true })
      render(<LogsToolbar {...props} />)

      expect(screen.getByTitle('Exit Fullscreen (F11)')).toBeInTheDocument()
    })

    it('should call onOpenSettings when Settings is clicked', async () => {
      const user = userEvent.setup()
      const onOpenSettings = vi.fn()
      const props = createDefaultProps({ onOpenSettings })
      render(<LogsToolbar {...props} />)

      const settingsButton = screen.getByTitle('Settings (Ctrl+,)')
      await user.click(settingsButton)

      expect(onOpenSettings).toHaveBeenCalled()
    })
  })

  describe('status indicators', () => {
    it('should show paused indicator when isPaused is true', () => {
      const props = createDefaultProps({ isPaused: true })
      render(<LogsToolbar {...props} />)

      expect(screen.getByText(/⏸ Paused/)).toBeInTheDocument()
    })

    it('should not show paused indicator when isPaused is false', () => {
      const props = createDefaultProps({ isPaused: false })
      render(<LogsToolbar {...props} />)

      expect(screen.queryByText(/⏸ Paused/)).not.toBeInTheDocument()
    })

    it('should show manual scroll indicator when autoScrollEnabled is false', () => {
      const props = createDefaultProps({ autoScrollEnabled: false })
      render(<LogsToolbar {...props} />)

      expect(screen.getByText(/↑ Manual scroll/)).toBeInTheDocument()
    })

    it('should not show manual scroll indicator when autoScrollEnabled is true', () => {
      const props = createDefaultProps({ autoScrollEnabled: true })
      render(<LogsToolbar {...props} />)

      expect(screen.queryByText(/↑ Manual scroll/)).not.toBeInTheDocument()
    })

    it('should show both indicators when both states are active', () => {
      const props = createDefaultProps({ isPaused: true, autoScrollEnabled: false })
      render(<LogsToolbar {...props} />)

      expect(screen.getByText(/⏸ Paused/)).toBeInTheDocument()
      expect(screen.getByText(/↑ Manual scroll/)).toBeInTheDocument()
    })

    it('should have correct aria-live attribute for accessibility', () => {
      const props = createDefaultProps({ isPaused: true })
      render(<LogsToolbar {...props} />)

      const pausedIndicator = screen.getByText(/⏸ Paused/)
      expect(pausedIndicator).toHaveAttribute('aria-live', 'polite')
    })
  })

  describe('edge cases', () => {
    it('should handle empty search term', () => {
      const props = createDefaultProps({ searchTerm: '' })
      render(<LogsToolbar {...props} />)

      const searchInput = screen.getByPlaceholderText('Search logs...')
      expect(searchInput).toHaveValue('')
    })

    it('should handle null bulkOperation', () => {
      const props = createDefaultProps({ bulkOperation: null })
      render(<LogsToolbar {...props} />)

      expect(screen.getByRole('toolbar')).toBeInTheDocument()
    })

    it('should handle undefined bulkOperation', () => {
      const props = createDefaultProps({ bulkOperation: undefined })
      render(<LogsToolbar {...props} />)

      expect(screen.getByRole('toolbar')).toBeInTheDocument()
    })

    it('should clamp gridColumns to valid range', () => {
      const props = createDefaultProps({ gridColumns: 1 })
      render(<LogsToolbar {...props} />)

      const decreaseButton = screen.getByRole('button', { name: /decrease columns/i })
      expect(decreaseButton).toBeDisabled()
    })
  })
})
