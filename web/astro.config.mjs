import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';
import mdx from '@astrojs/mdx';

// https://astro.build/config
export default defineConfig({
  site: 'https://jongio.github.io/azd-app/',
  base: '/azd-app/',
  integrations: [
    tailwind(),
    mdx()
  ],
  output: 'static'
});
