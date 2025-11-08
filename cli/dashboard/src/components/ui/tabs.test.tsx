import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

describe('Tabs', () => {
  describe('Tabs component', () => {
    it('should render tabs with children', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <div>Tab content</div>
        </Tabs>
      )
      
      expect(screen.getByText('Tab content')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      const { container } = render(
        <Tabs value="tab1" onValueChange={() => {}} className="custom-class">
          <div>Content</div>
        </Tabs>
      )
      
      const tabsContainer = container.firstChild
      expect(tabsContainer).toHaveClass('custom-class')
    })

    it('should provide context to children', () => {
      const onValueChange = vi.fn()
      render(
        <Tabs value="tab1" onValueChange={onValueChange}>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
        </Tabs>
      )
      
      expect(screen.getByText('Tab 1')).toBeInTheDocument()
    })

    it('should call onValueChange when value changes', async () => {
      const user = userEvent.setup()
      const onValueChange = vi.fn()
      
      render(
        <Tabs value="tab1" onValueChange={onValueChange}>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </Tabs>
      )
      
      await user.click(screen.getByText('Tab 2'))
      
      expect(onValueChange).toHaveBeenCalledWith('tab2')
    })
  })

  describe('TabsList component', () => {
    it('should render list with children', () => {
      render(
        <TabsList>
          <div>List content</div>
        </TabsList>
      )
      
      expect(screen.getByText('List content')).toBeInTheDocument()
    })

    it('should apply default styling classes', () => {
      const { container } = render(
        <TabsList>
          <div>Content</div>
        </TabsList>
      )
      
      const list = container.firstChild
      expect(list).toHaveClass('inline-flex', 'h-10', 'items-center', 'justify-center')
      expect(list).toHaveClass('rounded-md', 'bg-muted', 'p-1', 'text-muted-foreground')
    })

    it('should apply custom className', () => {
      const { container } = render(
        <TabsList className="custom-list-class">
          <div>Content</div>
        </TabsList>
      )
      
      const list = container.firstChild
      expect(list).toHaveClass('custom-list-class')
    })

    it('should render without custom className', () => {
      const { container } = render(
        <TabsList>
          <div>Content</div>
        </TabsList>
      )
      
      expect(container.firstChild).toBeInTheDocument()
    })
  })

  describe('TabsTrigger component', () => {
    it('should render trigger button', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
        </Tabs>
      )
      
      const button = screen.getByRole('button', { name: 'Tab 1' })
      expect(button).toBeInTheDocument()
    })

    it('should apply active styling when trigger is active', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsTrigger value="tab1">Active Tab</TabsTrigger>
        </Tabs>
      )
      
      const button = screen.getByRole('button', { name: 'Active Tab' })
      expect(button).toHaveClass('bg-background', 'text-foreground', 'shadow-sm')
    })

    it('should apply inactive styling when trigger is not active', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsTrigger value="tab2">Inactive Tab</TabsTrigger>
        </Tabs>
      )
      
      const button = screen.getByRole('button', { name: 'Inactive Tab' })
      expect(button).toHaveClass('hover:bg-background/50')
      expect(button).not.toHaveClass('shadow-sm')
    })

    it('should call onValueChange when clicked', async () => {
      const user = userEvent.setup()
      const onValueChange = vi.fn()
      
      render(
        <Tabs value="tab1" onValueChange={onValueChange}>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </Tabs>
      )
      
      await user.click(screen.getByRole('button', { name: 'Tab 2' }))
      
      expect(onValueChange).toHaveBeenCalledWith('tab2')
      expect(onValueChange).toHaveBeenCalledTimes(1)
    })

    it('should throw error when used outside Tabs context', () => {
      // Suppress console.error for this test
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
      
      expect(() => {
        render(<TabsTrigger value="tab1">Tab</TabsTrigger>)
      }).toThrow('TabsTrigger must be used within Tabs')
      
      consoleError.mockRestore()
    })

    it('should apply custom className', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsTrigger value="tab1" className="custom-trigger-class">
            Tab
          </TabsTrigger>
        </Tabs>
      )
      
      const button = screen.getByRole('button', { name: 'Tab' })
      expect(button).toHaveClass('custom-trigger-class')
    })

    it('should apply base styling classes', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsTrigger value="tab1">Tab</TabsTrigger>
        </Tabs>
      )
      
      const button = screen.getByRole('button', { name: 'Tab' })
      expect(button).toHaveClass('inline-flex', 'items-center', 'justify-center')
      expect(button).toHaveClass('whitespace-nowrap', 'rounded-sm', 'px-3', 'py-1.5')
      expect(button).toHaveClass('text-sm', 'font-medium', 'transition-all')
    })

    it('should handle multiple triggers', async () => {
      const user = userEvent.setup()
      const onValueChange = vi.fn()
      
      render(
        <Tabs value="tab1" onValueChange={onValueChange}>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
          <TabsTrigger value="tab3">Tab 3</TabsTrigger>
        </Tabs>
      )
      
      const tab1 = screen.getByRole('button', { name: 'Tab 1' })
      const tab2 = screen.getByRole('button', { name: 'Tab 2' })
      const tab3 = screen.getByRole('button', { name: 'Tab 3' })
      
      expect(tab1).toHaveClass('bg-background')
      expect(tab2).not.toHaveClass('bg-background')
      expect(tab3).not.toHaveClass('bg-background')
      
      await user.click(tab2)
      expect(onValueChange).toHaveBeenCalledWith('tab2')
    })
  })

  describe('TabsContent component', () => {
    it('should render content when value matches', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsContent value="tab1">Content for tab 1</TabsContent>
        </Tabs>
      )
      
      expect(screen.getByText('Content for tab 1')).toBeInTheDocument()
    })

    it('should not render content when value does not match', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsContent value="tab2">Content for tab 2</TabsContent>
        </Tabs>
      )
      
      expect(screen.queryByText('Content for tab 2')).not.toBeInTheDocument()
    })

    it('should throw error when used outside Tabs context', () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
      
      expect(() => {
        render(<TabsContent value="tab1">Content</TabsContent>)
      }).toThrow('TabsContent must be used within Tabs')
      
      consoleError.mockRestore()
    })

    it('should apply custom className when content is visible', () => {
      const { container } = render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsContent value="tab1" className="custom-content-class">
            Content
          </TabsContent>
        </Tabs>
      )
      
      const content = container.querySelector('.custom-content-class')
      expect(content).toBeInTheDocument()
    })

    it('should apply default styling classes', () => {
      const { container } = render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsContent value="tab1">Content</TabsContent>
        </Tabs>
      )
      
      const content = container.querySelector('.mt-2')
      expect(content).toBeInTheDocument()
    })

    it('should switch between different content panels', () => {
      const { rerender } = render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsContent value="tab1">Content 1</TabsContent>
          <TabsContent value="tab2">Content 2</TabsContent>
        </Tabs>
      )
      
      expect(screen.getByText('Content 1')).toBeInTheDocument()
      expect(screen.queryByText('Content 2')).not.toBeInTheDocument()
      
      rerender(
        <Tabs value="tab2" onValueChange={() => {}}>
          <TabsContent value="tab1">Content 1</TabsContent>
          <TabsContent value="tab2">Content 2</TabsContent>
        </Tabs>
      )
      
      expect(screen.queryByText('Content 1')).not.toBeInTheDocument()
      expect(screen.getByText('Content 2')).toBeInTheDocument()
    })
  })

  describe('Complete Tabs integration', () => {
    it('should render complete tabs component with all parts', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsList>
            <TabsTrigger value="tab1">Tab 1</TabsTrigger>
            <TabsTrigger value="tab2">Tab 2</TabsTrigger>
          </TabsList>
          <TabsContent value="tab1">Content for tab 1</TabsContent>
          <TabsContent value="tab2">Content for tab 2</TabsContent>
        </Tabs>
      )
      
      expect(screen.getByText('Tab 1')).toBeInTheDocument()
      expect(screen.getByText('Tab 2')).toBeInTheDocument()
      expect(screen.getByText('Content for tab 1')).toBeInTheDocument()
      expect(screen.queryByText('Content for tab 2')).not.toBeInTheDocument()
    })

    it('should switch content when clicking different tabs', async () => {
      const user = userEvent.setup()
      
      const TestComponent = () => {
        const [value, setValue] = React.useState('tab1')
        
        return (
          <Tabs value={value} onValueChange={setValue}>
            <TabsList>
              <TabsTrigger value="tab1">Tab 1</TabsTrigger>
              <TabsTrigger value="tab2">Tab 2</TabsTrigger>
            </TabsList>
            <TabsContent value="tab1">Content 1</TabsContent>
            <TabsContent value="tab2">Content 2</TabsContent>
          </Tabs>
        )
      }
      
      render(<TestComponent />)
      
      expect(screen.getByText('Content 1')).toBeInTheDocument()
      expect(screen.queryByText('Content 2')).not.toBeInTheDocument()
      
      await user.click(screen.getByText('Tab 2'))
      
      expect(screen.queryByText('Content 1')).not.toBeInTheDocument()
      expect(screen.getByText('Content 2')).toBeInTheDocument()
    })

    it('should handle empty children gracefully', () => {
      render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsList>
            <div></div>
          </TabsList>
        </Tabs>
      )
      
      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })

    it('should maintain styling consistency across all parts', () => {
      const { container } = render(
        <Tabs value="tab1" onValueChange={() => {}}>
          <TabsList>
            <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          </TabsList>
          <TabsContent value="tab1">Content</TabsContent>
        </Tabs>
      )
      
      const list = container.querySelector('.inline-flex')
      const trigger = screen.getByRole('button')
      const content = screen.getByText('Content').parentElement
      
      expect(list).toBeInTheDocument()
      expect(trigger).toBeInTheDocument()
      expect(content).toBeInTheDocument()
    })
  })
})
