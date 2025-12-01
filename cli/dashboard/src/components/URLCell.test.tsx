import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { URLCell } from '@/components/URLCell'

describe('URLCell', () => {
  it('should display dash when no URLs are provided', () => {
    render(<URLCell />)
    
    expect(screen.getByText('-')).toBeInTheDocument()
    expect(screen.getByText('-')).toHaveClass('text-muted-foreground')
  })

  it('should display local URL when only local is provided', () => {
    render(<URLCell localUrl="http://localhost:3000" />)
    
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'http://localhost:3000')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    expect(link).toHaveTextContent('http://localhost:3000')
  })

  it('should display Azure URL when only Azure is provided', () => {
    render(<URLCell azureUrl="https://my-app.azurewebsites.net" />)
    
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'https://my-app.azurewebsites.net')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveTextContent('https://my-app.azurewebsites.net')
  })

  it('should prefer local URL when both are provided', () => {
    render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    // Primary link should be local
    const links = screen.getAllByRole('link')
    expect(links[0]).toHaveAttribute('href', 'http://localhost:3000')
    expect(links[0]).toHaveTextContent('http://localhost:3000')
  })

  it('should show +1 badge when both URLs are provided', () => {
    render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    expect(screen.getByText('+1')).toBeInTheDocument()
  })

  it('should display Azure URL in tooltip when both URLs are provided', () => {
    render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    // Azure URL should be in the document (in the tooltip)
    const azureLink = screen.getAllByRole('link').find(link => 
      link.getAttribute('href') === 'https://my-app.azurewebsites.net'
    )
    expect(azureLink).toBeInTheDocument()
  })

  it('should truncate long URLs', () => {
    const longUrl = 'http://localhost:3000/very/long/path/that/exceeds/the/maximum/length/allowed'
    render(<URLCell localUrl={longUrl} />)
    
    const link = screen.getByRole('link')
    // URL should be truncated (contains ...)
    expect(link.textContent).toContain('...')
    // But href should be full URL
    expect(link).toHaveAttribute('href', longUrl)
  })

  it('should truncate URL while preserving protocol', () => {
    const longUrl = 'https://very-long-subdomain-name.azurewebsites.net/api/v1/endpoint/with/many/segments'
    render(<URLCell localUrl={longUrl} />)
    
    const link = screen.getByRole('link')
    expect(link.textContent).toContain('https://')
    expect(link.textContent).toContain('...')
  })

  it('should not truncate short URLs', () => {
    const shortUrl = 'http://localhost:3000'
    render(<URLCell localUrl={shortUrl} />)
    
    const link = screen.getByRole('link')
    expect(link.textContent).toBe(shortUrl)
    expect(link.textContent).not.toContain('...')
  })

  it('should handle URLs without protocol', () => {
    const urlWithoutProtocol = 'localhost:3000'
    render(<URLCell localUrl={urlWithoutProtocol} />)
    
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', urlWithoutProtocol)
  })

  it('should display ExternalLink icon', () => {
    const { container } = render(<URLCell localUrl="http://localhost:3000" />)
    
    // ExternalLink icon should be present
    const icon = container.querySelector('.lucide-external-link')
    expect(icon).toBeInTheDocument()
  })

  it('should have hover effects on primary link', () => {
    render(<URLCell localUrl="http://localhost:3000" />)
    
    const link = screen.getByRole('link')
    expect(link).toHaveClass('hover:underline')
    expect(link).toHaveClass('transition-colors')
  })

  it('should set title attribute on primary link', () => {
    const url = 'http://localhost:3000'
    render(<URLCell localUrl={url} />)
    
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('title', url)
  })

  it('should set title attribute on badge for multiple URLs', () => {
    render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    const badge = screen.getByText('+1')
    expect(badge).toHaveAttribute('title', 'Azure URL available')
  })

  it('should show Azure URL label in tooltip', () => {
    render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    expect(screen.getByText('Azure URL:')).toBeInTheDocument()
  })

  it('should have correct styling for Azure badge', () => {
    render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    const badge = screen.getByText('+1')
    expect(badge).toHaveClass('bg-primary/20', 'text-primary')
  })

  it('should have correct styling for tooltip container', () => {
    const { container } = render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    const tooltip = container.querySelector('.group-hover\\:block')
    expect(tooltip).toBeInTheDocument()
    expect(tooltip).toHaveClass('hidden')
  })

  it('should render Azure link with external icon', () => {
    const { container } = render(<URLCell localUrl="http://localhost:3000" azureUrl="https://my-app.azurewebsites.net" />)
    
    // Should have multiple ExternalLink icons
    const icons = container.querySelectorAll('.lucide-external-link')
    expect(icons.length).toBeGreaterThanOrEqual(2)
  })

  it('should handle very long Azure URLs in tooltip', () => {
    const longAzureUrl = 'https://very-long-application-name-with-many-segments.azurewebsites.net/api/v1'
    render(<URLCell localUrl="http://localhost:3000" azureUrl={longAzureUrl} />)
    
    const azureLink = screen.getAllByRole('link').find(link => 
      link.getAttribute('href') === longAzureUrl
    )
    expect(azureLink).toBeInTheDocument()
  })

  it('should apply truncate class to URLs', () => {
    const { container } = render(<URLCell localUrl="http://localhost:3000/some/path" />)
    
    const truncateElement = container.querySelector('.truncate')
    expect(truncateElement).toBeInTheDocument()
    expect(truncateElement).toHaveClass('max-w-[300px]')
  })

  it('should maintain accessibility with proper rel attribute', () => {
    render(<URLCell localUrl="http://localhost:3000" />)
    
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should not show +1 badge when only one URL is provided', () => {
    render(<URLCell localUrl="http://localhost:3000" />)
    
    expect(screen.queryByText('+1')).not.toBeInTheDocument()
  })

  it('should handle URLs at exactly 40 characters', () => {
    // URL exactly 40 chars: "http://localhost:3000/exactly40chars"
    const exactUrl = 'http://localhost:3000/path/exactly40c'
    render(<URLCell localUrl={exactUrl} />)
    
    const link = screen.getByRole('link')
    expect(link.textContent).not.toContain('...')
  })

  it('should truncate protocol-less URLs correctly', () => {
    const longUrlNoProtocol = 'a'.repeat(50)
    render(<URLCell localUrl={longUrlNoProtocol} />)
    
    const link = screen.getByRole('link')
    expect(link.textContent).toContain('...')
    expect(link.textContent.length).toBeLessThan(longUrlNoProtocol.length)
  })

  it('should handle edge case with empty string URLs', () => {
    render(<URLCell localUrl="" azureUrl="" />)
    
    expect(screen.getByText('-')).toBeInTheDocument()
  })

  it('should handle undefined vs not provided URLs', () => {
    render(<URLCell localUrl={undefined} azureUrl={undefined} />)
    
    expect(screen.getByText('-')).toBeInTheDocument()
  })

  it('should prioritize local URL even when Azure URL is longer', () => {
    const shortLocal = 'http://localhost:3000'
    const longAzure = 'https://very-long-azure-url.azurewebsites.net/with/many/segments'
    render(<URLCell localUrl={shortLocal} azureUrl={longAzure} />)
    
    const primaryLink = screen.getAllByRole('link')[0]
    expect(primaryLink).toHaveAttribute('href', shortLocal)
  })
})
