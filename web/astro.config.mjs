import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import mdx from '@astrojs/mdx';
import expressiveCode from 'astro-expressive-code';

// https://astro.build/config
export default defineConfig({
  site: 'https://jongio.github.io/azd-app/',
  base: '/azd-app/',
  integrations: [
    expressiveCode(),
    mdx()
  ],
  server: {
    host: '127.0.0.1',
    port: 4331,
  },
  preview: {
    host: '127.0.0.1',
    port: 4331,
  },
  vite: {
    plugins: [tailwindcss()]
  },
  output: 'static'
});
