import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
} from '@/components/ui/table'

describe('Table Components', () => {
  describe('Table', () => {
    it('should render table element', () => {
      render(
        <Table>
          <tbody>
            <tr>
              <td>Content</td>
            </tr>
          </tbody>
        </Table>
      )
      
      const table = document.querySelector('table')
      expect(table).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(
        <Table className="custom-class">
          <tbody>
            <tr>
              <td>Content</td>
            </tr>
          </tbody>
        </Table>
      )
      
      const table = document.querySelector('table')
      expect(table).toHaveClass('custom-class')
    })

    it('should have overflow wrapper', () => {
      const { container } = render(
        <Table>
          <tbody>
            <tr>
              <td>Content</td>
            </tr>
          </tbody>
        </Table>
      )
      
      const wrapper = container.querySelector('.overflow-auto')
      expect(wrapper).toBeInTheDocument()
    })
  })

  describe('TableHeader', () => {
    it('should render thead element', () => {
      render(
        <table>
          <TableHeader>
            <tr>
              <th>Header</th>
            </tr>
          </TableHeader>
        </table>
      )
      
      const thead = document.querySelector('thead')
      expect(thead).toBeInTheDocument()
    })

    it('should apply sticky positioning', () => {
      render(
        <table>
          <TableHeader>
            <tr>
              <th>Header</th>
            </tr>
          </TableHeader>
        </table>
      )
      
      const thead = document.querySelector('thead')
      expect(thead).toHaveClass('sticky', 'top-0')
    })
  })

  describe('TableBody', () => {
    it('should render tbody element', () => {
      render(
        <table>
          <TableBody>
            <tr>
              <td>Content</td>
            </tr>
          </TableBody>
        </table>
      )
      
      const tbody = document.querySelector('tbody')
      expect(tbody).toBeInTheDocument()
    })
  })

  describe('TableFooter', () => {
    it('should render tfoot element', () => {
      render(
        <table>
          <TableFooter>
            <tr>
              <td>Footer</td>
            </tr>
          </TableFooter>
        </table>
      )
      
      const tfoot = document.querySelector('tfoot')
      expect(tfoot).toBeInTheDocument()
    })

    it('should have border styling', () => {
      render(
        <table>
          <TableFooter>
            <tr>
              <td>Footer</td>
            </tr>
          </TableFooter>
        </table>
      )
      
      const tfoot = document.querySelector('tfoot')
      expect(tfoot).toHaveClass('border-t')
    })
  })

  describe('TableRow', () => {
    it('should render tr element', () => {
      render(
        <table>
          <tbody>
            <TableRow>
              <td>Content</td>
            </TableRow>
          </tbody>
        </table>
      )
      
      const tr = document.querySelector('tr')
      expect(tr).toBeInTheDocument()
    })

    it('should have hover styles', () => {
      render(
        <table>
          <tbody>
            <TableRow>
              <td>Content</td>
            </TableRow>
          </tbody>
        </table>
      )
      
      const tr = document.querySelector('tr')
      expect(tr).toHaveClass('hover:bg-white/5')
    })
  })

  describe('TableHead', () => {
    it('should render th element', () => {
      render(
        <table>
          <thead>
            <tr>
              <TableHead>Column</TableHead>
            </tr>
          </thead>
        </table>
      )
      
      expect(screen.getByText('Column')).toBeInTheDocument()
    })

    it('should have proper text alignment', () => {
      render(
        <table>
          <thead>
            <tr>
              <TableHead>Column</TableHead>
            </tr>
          </thead>
        </table>
      )
      
      const th = screen.getByText('Column')
      expect(th).toHaveClass('text-left')
    })
  })

  describe('TableCell', () => {
    it('should render td element', () => {
      render(
        <table>
          <tbody>
            <tr>
              <TableCell>Cell content</TableCell>
            </tr>
          </tbody>
        </table>
      )
      
      expect(screen.getByText('Cell content')).toBeInTheDocument()
    })

    it('should have padding', () => {
      render(
        <table>
          <tbody>
            <tr>
              <TableCell>Cell content</TableCell>
            </tr>
          </tbody>
        </table>
      )
      
      const td = screen.getByText('Cell content')
      expect(td).toHaveClass('p-4')
    })
  })

  describe('TableCaption', () => {
    it('should render caption element', () => {
      render(
        <table>
          <TableCaption>Table caption</TableCaption>
          <tbody>
            <tr>
              <td>Content</td>
            </tr>
          </tbody>
        </table>
      )
      
      expect(screen.getByText('Table caption')).toBeInTheDocument()
    })

    it('should have muted text color', () => {
      render(
        <table>
          <TableCaption>Table caption</TableCaption>
          <tbody>
            <tr>
              <td>Content</td>
            </tr>
          </tbody>
        </table>
      )
      
      const caption = screen.getByText('Table caption')
      expect(caption).toHaveClass('text-muted-foreground')
    })
  })

  describe('Complete Table', () => {
    it('should render complete table with all components', () => {
      render(
        <Table>
          <TableCaption>A list of services</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>Service 1</TableCell>
              <TableCell>Running</TableCell>
            </TableRow>
            <TableRow>
              <TableCell>Service 2</TableCell>
              <TableCell>Stopped</TableCell>
            </TableRow>
          </TableBody>
          <TableFooter>
            <TableRow>
              <TableCell>Total</TableCell>
              <TableCell>2 services</TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      )
      
      expect(screen.getByText('A list of services')).toBeInTheDocument()
      expect(screen.getByText('Name')).toBeInTheDocument()
      expect(screen.getByText('Status')).toBeInTheDocument()
      expect(screen.getByText('Service 1')).toBeInTheDocument()
      expect(screen.getByText('Service 2')).toBeInTheDocument()
      expect(screen.getByText('Running')).toBeInTheDocument()
      expect(screen.getByText('Stopped')).toBeInTheDocument()
      expect(screen.getByText('Total')).toBeInTheDocument()
      expect(screen.getByText('2 services')).toBeInTheDocument()
    })

    it('should apply custom classNames to all components', () => {
      render(
        <Table className="table-custom">
          <TableHeader className="header-custom">
            <TableRow className="row-custom">
              <TableHead className="head-custom">Name</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody className="body-custom">
            <TableRow>
              <TableCell className="cell-custom">Data</TableCell>
            </TableRow>
          </TableBody>
          <TableFooter className="footer-custom">
            <TableRow>
              <TableCell>Footer</TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      )
      
      const table = document.querySelector('table')
      const thead = document.querySelector('thead')
      const tbody = document.querySelector('tbody')
      const tfoot = document.querySelector('tfoot')
      const head = screen.getByText('Name')
      const cell = screen.getByText('Data')
      
      expect(table).toHaveClass('table-custom')
      expect(thead).toHaveClass('header-custom')
      expect(tbody).toHaveClass('body-custom')
      expect(tfoot).toHaveClass('footer-custom')
      expect(head).toHaveClass('head-custom')
      expect(cell).toHaveClass('cell-custom')
    })
  })
})
