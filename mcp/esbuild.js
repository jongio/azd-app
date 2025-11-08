#!/usr/bin/env node

/**
 * Bundle the MCP server into a single executable file with all dependencies.
 */

import * as esbuild from 'esbuild';

await esbuild.build({
  entryPoints: ['dist/index.js'],
  bundle: true,
  platform: 'node',
  target: 'node18',
  outfile: 'dist/index.bundle.js',
  format: 'esm',
  external: [],
  minify: false,
  sourcemap: false,
});

console.log('✅ MCP server bundled successfully!');
