/**
 * TimeRangeSelector Component Tests
 * 
 * Tests the time range selector component including preset selection,
 * custom range input, validation, and accessibility.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TimeRangeSelector, type TimeRangeSelectorProps } from './TimeRangeSelector'
import type { TimeRange } from '@/hooks/useHistoricalLogs'

describe('TimeRangeSelector', () => {
  const defaultProps: TimeRangeSelectorProps = {
    value: { preset: '15m' },
    onChange: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Basic Rendering', () => {
    it('renders with default 15m preset selected', () => {
      render(<TimeRangeSelector {...defaultProps} />)
      
      const preset15m = screen.getByRole('radio', { name: '15m' })
      expect(preset15m).toHaveAttribute('aria-checked', 'true')
    })

    it('renders all preset options', () => {
      render(<TimeRangeSelector {...defaultProps} />)
      
      expect(screen.getByRole('radio', { name: '15m' })).toBeInTheDocument()
      expect(screen.getByRole('radio', { name: '30m' })).toBeInTheDocument()
      expect(screen.getByRole('radio', { name: '6h' })).toBeInTheDocument()
      expect(screen.getByRole('radio', { name: '24h' })).toBeInTheDocument()
      expect(screen.getByRole('radio', { name: 'Custom' })).toBeInTheDocument()
    })

    it('shows label', () => {
      render(<TimeRangeSelector {...defaultProps} />)
      
      expect(screen.getByText('Time Range')).toBeInTheDocument()
    })
  })

  describe('Preset Selection', () => {
    it('calls onChange when 30m preset is clicked', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const preset30m = screen.getByRole('radio', { name: '30m' })
      await user.click(preset30m)
      
      expect(onChange).toHaveBeenCalledWith({ preset: '30m' })
    })

    it('calls onChange when 6h preset is clicked', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const preset6h = screen.getByRole('radio', { name: '6h' })
      await user.click(preset6h)
      
      expect(onChange).toHaveBeenCalledWith({ preset: '6h' })
    })

    it('calls onChange when 24h preset is clicked', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const preset24h = screen.getByRole('radio', { name: '24h' })
      await user.click(preset24h)
      
      expect(onChange).toHaveBeenCalledWith({ preset: '24h' })
    })

    it('shows selected state for active preset', () => {
      render(<TimeRangeSelector {...defaultProps} value={{ preset: '6h' }} />)
      
      const preset6h = screen.getByRole('radio', { name: '6h' })
      expect(preset6h).toHaveAttribute('aria-checked', 'true')
    })

    it('does not show custom range picker for standard presets', () => {
      render(<TimeRangeSelector {...defaultProps} value={{ preset: '30m' }} />)
      
      expect(screen.queryByLabelText(/Start/i)).not.toBeInTheDocument()
      expect(screen.queryByLabelText(/End/i)).not.toBeInTheDocument()
    })
  })

  describe('Custom Range Selection', () => {
    it('shows custom range picker when Custom is selected', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      expect(screen.getByLabelText(/Start/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/End/i)).toBeInTheDocument()
    })

    it('calls onChange with default start/end when Custom clicked', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          preset: 'custom',
          start: expect.any(Date),
          end: expect.any(Date),
        })
      )
    })

    it('shows datetime inputs for custom range', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const startInput = screen.getByLabelText(/Start/i)
      const endInput = screen.getByLabelText(/End/i)
      
      expect(startInput).toHaveAttribute('type', 'datetime-local')
      expect(endInput).toHaveAttribute('type', 'datetime-local')
    })

    it('shows Apply Range button in custom mode', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      expect(screen.getByRole('button', { name: /Apply Range/i })).toBeInTheDocument()
    })

    it('calls onChange when Apply Range is clicked', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      onChange.mockClear()
      
      const applyButton = screen.getByRole('button', { name: /Apply Range/i })
      await user.click(applyButton)
      
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          preset: 'custom',
          start: expect.any(Date),
          end: expect.any(Date),
        })
      )
    })

    it('updates start date when changed', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const startInput = screen.getByLabelText(/Start/i) as HTMLInputElement
      await user.clear(startInput)
      await user.type(startInput, '2024-01-15T10:00')
      
      expect(startInput.value).toBe('2024-01-15T10:00')
    })

    it('updates end date when changed', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const endInput = screen.getByLabelText(/End/i) as HTMLInputElement
      await user.clear(endInput)
      await user.type(endInput, '2024-01-15T12:00')
      
      expect(endInput.value).toBe('2024-01-15T12:00')
    })
  })

  describe('Range Validation', () => {
    it('swaps start and end if start is after end', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      const start = new Date('2024-01-15T12:00')
      const end = new Date('2024-01-15T10:00') // Earlier than start
      
      render(
        <TimeRangeSelector 
          {...defaultProps} 
          value={{ preset: 'custom', start, end }} 
          onChange={onChange} 
        />
      )
      
      const applyButton = screen.getByRole('button', { name: /Apply Range/i })
      await user.click(applyButton)
      
      // Should swap them
      expect(onChange).toHaveBeenCalledWith({
        preset: 'custom',
        start: end,
        end: start,
      })
    })

    it('shows maximum range hint', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      expect(screen.getByText('Maximum range: 7 days')).toBeInTheDocument()
    })

    it('clamps range to 7 days if exceeded', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      const end = new Date('2024-01-15T12:00')
      const start = new Date('2024-01-01T12:00') // 14 days earlier
      
      render(
        <TimeRangeSelector 
          {...defaultProps} 
          value={{ preset: 'custom', start, end }} 
          onChange={onChange} 
        />
      )
      
      const applyButton = screen.getByRole('button', { name: /Apply Range/i })
      await user.click(applyButton)
      
      // Should clamp to 7 days from end
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          preset: 'custom',
          end,
          start: expect.any(Date),
        })
      )
      
      const call = onChange.mock.calls[0][0] as TimeRange
      const diffDays = (call.end!.getTime() - call.start!.getTime()) / (1000 * 60 * 60 * 24)
      expect(diffDays).toBeLessThanOrEqual(7)
    })

    it('disables Apply button when dates are missing', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const startInput = screen.getByLabelText(/Start/i)
      await user.clear(startInput)
      
      const applyButton = screen.getByRole('button', { name: /Apply Range/i })
      expect(applyButton).toBeDisabled()
    })
  })

  describe('Disabled State', () => {
    it('disables all preset buttons when disabled', () => {
      render(<TimeRangeSelector {...defaultProps} disabled={true} />)
      
      const preset15m = screen.getByRole('radio', { name: '15m' })
      const preset30m = screen.getByRole('radio', { name: '30m' })
      
      expect(preset15m).toBeDisabled()
      expect(preset30m).toBeDisabled()
    })

    it('disables custom range inputs when disabled', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      render(<TimeRangeSelector {...defaultProps} value={{ preset: 'custom' }} disabled={true} />)
      
      const startInput = screen.getByLabelText(/Start/i)
      const endInput = screen.getByLabelText(/End/i)
      const applyButton = screen.getByRole('button', { name: /Apply Range/i })
      
      expect(startInput).toBeDisabled()
      expect(endInput).toBeDisabled()
      expect(applyButton).toBeDisabled()
    })
  })

  describe('Date Constraints', () => {
    it('sets min date to 7 days ago', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const startInput = screen.getByLabelText(/Start/i)
      expect(startInput).toHaveAttribute('min')
    })

    it('sets max date to now', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const endInput = screen.getByLabelText(/End/i)
      expect(endInput).toHaveAttribute('max')
    })

    it('constrains end date by start date', async () => {
      const start = new Date('2024-01-15T10:00')
      const end = new Date('2024-01-15T12:00')
      
      render(
        <TimeRangeSelector 
          {...defaultProps} 
          value={{ preset: 'custom', start, end }} 
        />
      )
      
      const endInput = screen.getByLabelText(/End/i)
      expect(endInput).toHaveAttribute('min')
    })
  })

  describe('Accessibility', () => {
    it('has radiogroup role for presets', () => {
      render(<TimeRangeSelector {...defaultProps} />)
      
      const radiogroup = screen.getByRole('radiogroup', { name: 'Time range selection' })
      expect(radiogroup).toBeInTheDocument()
    })

    it('has accessible labels for date inputs', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      expect(screen.getByLabelText(/Start/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/End/i)).toBeInTheDocument()
    })

    it('has accessible button labels', async () => {
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      expect(screen.getByRole('button', { name: /Apply Range/i })).toBeInTheDocument()
    })
  })

  describe('Custom Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <TimeRangeSelector {...defaultProps} className="custom-range-class" />
      )
      
      const element = container.querySelector('.custom-range-class')
      expect(element).toBeInTheDocument()
    })
  })

  describe('External Value Changes', () => {
    it('updates custom inputs when value changes externally', () => {
      const start = new Date('2024-01-15T10:00')
      const end = new Date('2024-01-15T12:00')
      
      const { rerender } = render(
        <TimeRangeSelector {...defaultProps} value={{ preset: 'custom', start, end }} />
      )
      
      const newStart = new Date('2024-01-16T10:00')
      const newEnd = new Date('2024-01-16T12:00')
      
      rerender(
        <TimeRangeSelector 
          {...defaultProps} 
          value={{ preset: 'custom', start: newStart, end: newEnd }} 
        />
      )
      
      const startInput = screen.getByLabelText(/Start/i) as HTMLInputElement
      expect(startInput.value).toContain('2024-01-16')
    })
  })

  describe('Edge Cases', () => {
    it('handles preset change while in custom mode', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      // Switch to custom
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      // Switch back to preset
      const preset30m = screen.getByRole('radio', { name: '30m' })
      await user.click(preset30m)
      
      expect(onChange).toHaveBeenLastCalledWith({ preset: '30m' })
      
      // Custom picker should be hidden
      expect(screen.queryByLabelText(/Start/i)).not.toBeInTheDocument()
    })

    it('initializes with sensible defaults for custom range', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TimeRangeSelector {...defaultProps} onChange={onChange} />)
      
      const customButton = screen.getByRole('radio', { name: 'Custom' })
      await user.click(customButton)
      
      const call = onChange.mock.calls[0][0] as TimeRange
      expect(call.preset).toBe('custom')
      expect(call.start).toBeInstanceOf(Date)
      expect(call.end).toBeInstanceOf(Date)
      
      // Default should be 1 hour ago to now
      const diffMs = call.end!.getTime() - call.start!.getTime()
      const diffHours = diffMs / (1000 * 60 * 60)
      expect(diffHours).toBeCloseTo(1, 0)
    })
  })
})
