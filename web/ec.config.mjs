import { defineEcConfig } from 'astro-expressive-code';

export default defineEcConfig({
  themes: ['github-light', 'github-dark'],
  themeCssSelector: (theme) => `[data-theme="${theme.type}"]`,
  styleOverrides: {
    borderRadius: '0.5rem',
    codePaddingBlock: '1rem',
    codePaddingInline: '1.25rem',
  },
  frames: {
    showCopyToClipboardButton: true,
  },
});
