import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

// Schema for CLI command reference pages
const commandsCollection = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/commands' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    command: z.string(),
    category: z.enum(['core', 'dev', 'info', 'config']).default('core'),
    order: z.number().default(0),
  }),
});

// Schema for guided tour steps
const tourCollection = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/tour' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    step: z.number(),
    screenshot: z.string().optional(),
    nextStep: z.string().optional(),
    prevStep: z.string().optional(),
  }),
});

export const collections = {
  commands: commandsCollection,
  tour: tourCollection,
};
