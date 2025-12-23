/**
 * search-highlight - Utility to highlight search terms in log messages
 */
import type { ReactNode } from 'react'

/**
 * Highlights occurrences of a search term in text
 * @param text - The text to search within
 * @param searchTerm - The term to highlight (case-insensitive)
 * @returns React nodes with highlighted matches
 */
export function highlightSearchTerm(text: string, searchTerm: string): ReactNode {
  if (!searchTerm.trim()) {
    return text
  }

  // Escape special regex characters in the search term
  const escapedTerm = searchTerm.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  
  // Create case-insensitive regex with global flag
  const regex = new RegExp(`(${escapedTerm})`, 'gi')
  
  // Check if there's a match first
  if (!regex.test(text)) {
    return text
  }
  
  // Reset regex lastIndex after test
  regex.lastIndex = 0
  
  // Split text by matches
  const parts = text.split(regex)
  
  return parts.map((part, index) => {
    // Check if this part matches the search term (case-insensitive)
    if (part.toLowerCase() === searchTerm.toLowerCase()) {
      return (
        <mark
          key={index}
          className="bg-yellow-200 dark:bg-yellow-500/40 text-slate-900 dark:text-slate-100 font-semibold rounded px-0.5"
        >
          {part}
        </mark>
      )
    }
    return part
  })
}

/**
 * Highlights search term in HTML content (for ANSI-converted logs)
 * @param htmlString - The HTML string to process
 * @param searchTerm - The term to highlight (case-insensitive)
 * @returns HTML string with highlighted matches
 */
export function highlightSearchTermInHtml(htmlString: string, searchTerm: string): string {
  if (!searchTerm.trim()) {
    return htmlString
  }

  // Escape special regex characters in the search term
  const escapedTerm = searchTerm.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  
  // Create a temporary div to parse HTML
  const tempDiv = document.createElement('div')
  tempDiv.innerHTML = htmlString
  
  // Function to highlight text in text nodes
  const highlightTextNodes = (node: Node) => {
    if (node.nodeType === Node.TEXT_NODE) {
      const text = node.textContent ?? ''
      const regex = new RegExp(`(${escapedTerm})`, 'gi')
      
      if (regex.test(text)) {
        const span = document.createElement('span')
        const parts = text.split(regex)
        
        parts.forEach((part) => {
          if (part.toLowerCase() === searchTerm.toLowerCase()) {
            const mark = document.createElement('mark')
            mark.className = 'bg-yellow-200 dark:bg-yellow-500/40 text-slate-900 dark:text-slate-100 font-semibold rounded px-0.5'
            mark.textContent = part
            span.appendChild(mark)
          } else {
            span.appendChild(document.createTextNode(part))
          }
        })
        
        node.parentNode?.replaceChild(span, node)
      }
    } else if (node.nodeType === Node.ELEMENT_NODE) {
      // Don't process children of mark elements to avoid double-highlighting
      if ((node as Element).tagName !== 'MARK') {
        Array.from(node.childNodes).forEach(highlightTextNodes)
      }
    }
  }
  
  highlightTextNodes(tempDiv)
  
  return tempDiv.innerHTML
}
