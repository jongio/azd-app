import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { highlightSearchTerm, highlightSearchTermInHtml } from './search-highlight'

describe('highlightSearchTerm', () => {
  it('returns plain text when search term is empty', () => {
    const result = highlightSearchTerm('Hello World', '')
    expect(result).toBe('Hello World')
  })

  it('returns plain text when search term is only whitespace', () => {
    const result = highlightSearchTerm('Hello World', '   ')
    expect(result).toBe('Hello World')
  })

  it('highlights a single match (case-insensitive)', () => {
    const result = highlightSearchTerm('Hello World', 'world')
    const { container } = render(<>{result}</>)
    
    const mark = container.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('World')
    expect(mark?.className).toContain('bg-yellow-200')
    expect(mark?.className).toContain('dark:bg-yellow-500/40')
  })

  it('highlights multiple matches', () => {
    const result = highlightSearchTerm('Error: Failed to connect. Error code 500', 'error')
    const { container } = render(<>{result}</>)
    
    const marks = container.querySelectorAll('mark')
    expect(marks.length).toBe(2)
    expect(marks[0]?.textContent).toBe('Error')
    expect(marks[1]?.textContent).toBe('Error')
  })

  it('preserves original case in highlighted text', () => {
    const result = highlightSearchTerm('ERROR: An Error Occurred', 'error')
    const { container } = render(<>{result}</>)
    
    const marks = container.querySelectorAll('mark')
    expect(marks.length).toBe(2)
    expect(marks[0]?.textContent).toBe('ERROR')
    expect(marks[1]?.textContent).toBe('Error')
  })

  it('handles special regex characters in search term', () => {
    const result = highlightSearchTerm('Price: $100.50 (tax)', '$100.50')
    const { container } = render(<>{result}</>)
    
    const mark = container.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('$100.50')
  })

  it('handles parentheses in search term', () => {
    const result = highlightSearchTerm('Function call: func(arg)', 'func(arg)')
    const { container } = render(<>{result}</>)
    
    const mark = container.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('func(arg)')
  })

  it('handles square brackets in search term', () => {
    const result = highlightSearchTerm('Array: [1, 2, 3]', '[1, 2, 3]')
    const { container } = render(<>{result}</>)
    
    const mark = container.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('[1, 2, 3]')
  })

  it('returns text parts without highlighting when no match', () => {
    const result = highlightSearchTerm('Hello World', 'xyz')
    expect(result).toBe('Hello World')
  })

  it('handles partial word matches', () => {
    const result = highlightSearchTerm('Testing testability', 'test')
    const { container } = render(<>{result}</>)
    
    const marks = container.querySelectorAll('mark')
    expect(marks.length).toBe(2)
    expect(marks[0]?.textContent).toBe('Test')
    expect(marks[1]?.textContent).toBe('test')
  })
})

describe('highlightSearchTermInHtml', () => {
  beforeEach(() => {
    // Setup before each test
  })

  afterEach(() => {
    // Cleanup after each test
  })

  it('returns plain HTML when search term is empty', () => {
    const html = '<span class="text-red">Error occurred</span>'
    const result = highlightSearchTermInHtml(html, '')
    expect(result).toBe(html)
  })

  it('returns plain HTML when search term is only whitespace', () => {
    const html = '<span class="text-red">Error occurred</span>'
    const result = highlightSearchTermInHtml(html, '   ')
    expect(result).toBe(html)
  })

  it('highlights text inside HTML elements', () => {
    const html = '<span class="text-red">Error occurred</span>'
    const result = highlightSearchTermInHtml(html, 'error')
    
    expect(result).toContain('<mark')
    expect(result).toContain('bg-yellow-200')
    expect(result).toContain('Error')
  })

  it('preserves HTML structure while highlighting', () => {
    const html = '<span class="color-red">Server error</span><span> detected</span>'
    const result = highlightSearchTermInHtml(html, 'error')
    
    // Should still have two span elements
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const spans = tempDiv.querySelectorAll('span')
    expect(spans.length).toBeGreaterThanOrEqual(2)
    
    // Should have a mark element for highlighting
    const mark = tempDiv.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('error')
  })

  it('highlights multiple occurrences in HTML', () => {
    const html = '<p>Error in line 1</p><p>Another error in line 2</p>'
    const result = highlightSearchTermInHtml(html, 'error')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const marks = tempDiv.querySelectorAll('mark')
    expect(marks.length).toBe(2)
  })

  it('handles ANSI color codes (as HTML)', () => {
    const html = '<span style="color: red">ERROR:</span> Connection failed'
    const result = highlightSearchTermInHtml(html, 'error')
    
    expect(result).toContain('<mark')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const mark = tempDiv.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('ERROR')
  })

  it('handles special regex characters in search term', () => {
    const html = '<span>Price: $100.50</span>'
    const result = highlightSearchTermInHtml(html, '$100.50')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const mark = tempDiv.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('$100.50')
  })

  it('does not double-highlight already marked text', () => {
    const html = '<mark class="existing">error</mark> and another error'
    const result = highlightSearchTermInHtml(html, 'error')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const marks = tempDiv.querySelectorAll('mark')
    // Should have the original mark plus one new mark
    expect(marks.length).toBe(2)
  })

  it('is case-insensitive', () => {
    const html = '<p>ERROR: An Error occurred</p>'
    const result = highlightSearchTermInHtml(html, 'error')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const marks = tempDiv.querySelectorAll('mark')
    expect(marks.length).toBe(2)
    expect(marks[0]?.textContent).toBe('ERROR')
    expect(marks[1]?.textContent).toBe('Error')
  })

  it('handles nested HTML elements', () => {
    const html = '<div><span>Hello <strong>World</strong></span></div>'
    const result = highlightSearchTermInHtml(html, 'world')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const mark = tempDiv.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('World')
  })

  it('handles plain text nodes alongside element nodes', () => {
    const html = 'Plain text <span>and span text</span> more plain'
    const result = highlightSearchTermInHtml(html, 'text')
    
    const tempDiv = document.createElement('div')
    tempDiv.innerHTML = result
    
    const marks = tempDiv.querySelectorAll('mark')
    expect(marks.length).toBe(2)
  })
})

describe('search highlighting integration', () => {
  it('highlights search term in log message rendered to screen', () => {
    const logMessage = 'Application started successfully on port 3000'
    const searchTerm = 'port'
    
    const highlighted = highlightSearchTerm(logMessage, searchTerm)
    render(<div>{highlighted}</div>)
    
    const mark = screen.getByText('port')
    expect(mark.tagName).toBe('MARK')
    expect(mark.className).toContain('bg-yellow-200')
  })

  it('highlights search term in HTML log message', () => {
    const htmlMessage = '<span style="color: blue">Server listening on port 8080</span>'
    const searchTerm = 'listening'
    
    const highlighted = highlightSearchTermInHtml(htmlMessage, searchTerm)
    const { container } = render(<div dangerouslySetInnerHTML={{ __html: highlighted }} />)
    
    const mark = container.querySelector('mark')
    expect(mark).toBeTruthy()
    expect(mark?.textContent).toBe('listening')
    expect(mark?.className).toContain('bg-yellow-200')
  })

  it('works with real-world log examples', () => {
    const logs = [
      'ERROR: Database connection failed',
      'WARN: Retrying connection in 5 seconds',
      'INFO: Connection established',
      'ERROR: Query timeout exceeded'
    ]
    
    const searchTerm = 'connection'
    
    logs.forEach((log) => {
      const highlighted = highlightSearchTerm(log, searchTerm)
      const { container } = render(<div>{highlighted}</div>)
      
      const mark = container.querySelector('mark')
      if (log.toLowerCase().includes(searchTerm.toLowerCase())) {
        expect(mark).toBeTruthy()
      }
    })
  })
})
