import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import AnsiConverter from 'ansi-to-html'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const ansiConverter = new AnsiConverter({
  fg: '#d4d4d4',
  bg: '#0d0d0d',
  newline: false,
  escapeXML: true,
})

export function convertAnsiToHtml(text: string): string {
  return ansiConverter.toHtml(text)
}
