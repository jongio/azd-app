import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuGroup,
  DropdownMenuRadioGroup,
} from './dropdown-menu'

describe('DropdownMenu', () => {
  describe('basic rendering', () => {
    it('should render dropdown trigger', () => {
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      expect(screen.getByText('Open Menu')).toBeInTheDocument()
    })

    it('should open dropdown when trigger is clicked', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))

      expect(screen.getByText('Item 1')).toBeInTheDocument()
    })

    it('should close dropdown when pressing Escape', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('Item 1')).toBeInTheDocument()

      await user.keyboard('{Escape}')
      // After closing, item should not be visible
      await waitFor(() => {
        expect(screen.queryByText('Item 1')).not.toBeInTheDocument()
      })
    })
  })

  describe('DropdownMenuItem', () => {
    it('should call onClick when item is clicked', async () => {
      const user = userEvent.setup()
      const onClick = vi.fn()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={onClick}>Click Me</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      await user.click(screen.getByText('Click Me'))

      expect(onClick).toHaveBeenCalled()
    })

    it('should apply inset class when inset prop is true', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem inset>Inset Item</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const item = screen.getByText('Inset Item')
      expect(item).toHaveClass('pl-8')
    })

    it('should apply disabled state', async () => {
      const user = userEvent.setup()
      const onClick = vi.fn()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem disabled onClick={onClick}>
              Disabled Item
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const item = screen.getByText('Disabled Item')
      expect(item).toHaveAttribute('data-disabled')
    })
  })

  describe('DropdownMenuLabel', () => {
    it('should render label text', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuLabel>My Label</DropdownMenuLabel>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('My Label')).toBeInTheDocument()
    })

    it('should apply inset class when inset prop is true', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuLabel inset>Inset Label</DropdownMenuLabel>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const label = screen.getByText('Inset Label')
      expect(label).toHaveClass('pl-8')
    })
  })

  describe('DropdownMenuSeparator', () => {
    it('should render separator', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
            <DropdownMenuSeparator data-testid="separator" />
            <DropdownMenuItem>Item 2</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByTestId('separator')).toBeInTheDocument()
    })
  })

  describe('DropdownMenuShortcut', () => {
    it('should render keyboard shortcut text', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>
              Save
              <DropdownMenuShortcut>Ctrl+S</DropdownMenuShortcut>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('Ctrl+S')).toBeInTheDocument()
    })

    it('should have correct styling classes', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>
              Save
              <DropdownMenuShortcut>Ctrl+S</DropdownMenuShortcut>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const shortcut = screen.getByText('Ctrl+S')
      expect(shortcut).toHaveClass('ml-auto')
      expect(shortcut).toHaveClass('text-xs')
    })
  })

  describe('DropdownMenuGroup', () => {
    it('should group menu items', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuGroup>
              <DropdownMenuItem>Item 1</DropdownMenuItem>
              <DropdownMenuItem>Item 2</DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('Item 1')).toBeInTheDocument()
      expect(screen.getByText('Item 2')).toBeInTheDocument()
    })
  })

  describe('DropdownMenuCheckboxItem', () => {
    it('should render checkbox item', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuCheckboxItem checked>
              Checked Item
            </DropdownMenuCheckboxItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('Checked Item')).toBeInTheDocument()
    })

    it('should show check indicator when checked', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuCheckboxItem checked data-testid="checkbox-item">
              Checked Item
            </DropdownMenuCheckboxItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const item = screen.getByTestId('checkbox-item')
      expect(item).toHaveAttribute('data-state', 'checked')
    })
  })

  describe('DropdownMenuRadioGroup and DropdownMenuRadioItem', () => {
    it('should render radio items within group', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuRadioGroup value="option1">
              <DropdownMenuRadioItem value="option1">Option 1</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="option2">Option 2</DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('Option 1')).toBeInTheDocument()
      expect(screen.getByText('Option 2')).toBeInTheDocument()
    })

    it('should indicate selected radio item', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuRadioGroup value="option1">
              <DropdownMenuRadioItem value="option1" data-testid="radio1">
                Option 1
              </DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="option2" data-testid="radio2">
                Option 2
              </DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByTestId('radio1')).toHaveAttribute('data-state', 'checked')
      expect(screen.getByTestId('radio2')).toHaveAttribute('data-state', 'unchecked')
    })
  })

  describe('accessibility', () => {
    it('should have proper role attributes', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const content = screen.getByRole('menu')
      expect(content).toBeInTheDocument()
    })

    it('should have menuitem role for items', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      const item = screen.getByRole('menuitem')
      expect(item).toBeInTheDocument()
    })

    it('should support keyboard navigation', async () => {
      const user = userEvent.setup()
      const onClick = vi.fn()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={onClick}>Item 1</DropdownMenuItem>
            <DropdownMenuItem>Item 2</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      // Open with keyboard
      const trigger = screen.getByText('Open Menu')
      trigger.focus()
      await user.keyboard('{Enter}')

      expect(screen.getByText('Item 1')).toBeInTheDocument()
    })
  })

  describe('custom className', () => {
    it('should apply custom className to DropdownMenuContent', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent className="custom-content" data-testid="content">
            <DropdownMenuItem>Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByTestId('content')).toHaveClass('custom-content')
    })

    it('should apply custom className to DropdownMenuItem', async () => {
      const user = userEvent.setup()
      render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open Menu</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem className="custom-item">Item 1</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )

      await user.click(screen.getByText('Open Menu'))
      expect(screen.getByText('Item 1')).toHaveClass('custom-item')
    })
  })
})
