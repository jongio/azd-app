import { defineCollection, z } from 'astro:content';

// Schema for CLI command reference pages
const commandsCollection = defineCollection({
  type: 'content',
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
  type: 'content',
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
