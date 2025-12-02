/**
 * Tour Steps Data
 * 
 * Defines the 8-step guided tour structure with all metadata.
 */

export interface TourStep {
  /** Step number (1-8) */
  number: number;
  /** URL slug */
  slug: string;
  /** Step title */
  title: string;
  /** Short description */
  description: string;
  /** Estimated completion time in minutes */
  estimatedTime: number;
  /** Step icon/emoji */
  icon: string;
  /** Prerequisites (previous step numbers) */
  prerequisites: number[];
  /** Related documentation links */
  relatedDocs: string[];
}

export const tourSteps: TourStep[] = [
  {
    number: 1,
    slug: "1-install",
    title: "Install azd + extension",
    description: "Set up the Azure Developer CLI and install the azd-app extension",
    estimatedTime: 5,
    icon: "📦",
    prerequisites: [],
    relatedDocs: ["/docs/installation"],
  },
  {
    number: 2,
    slug: "2-requirements",
    title: "Check requirements",
    description: "Verify your system meets all requirements for running azd-app",
    estimatedTime: 3,
    icon: "✅",
    prerequisites: [1],
    relatedDocs: ["/docs/requirements"],
  },
  {
    number: 3,
    slug: "3-dependencies",
    title: "Install dependencies",
    description: "Install required dependencies and configure your development environment",
    estimatedTime: 5,
    icon: "🔧",
    prerequisites: [1, 2],
    relatedDocs: ["/docs/dependencies"],
  },
  {
    number: 4,
    slug: "4-first-app",
    title: "Run your first app",
    description: "Initialize and run a demo application using azd-app",
    estimatedTime: 7,
    icon: "🚀",
    prerequisites: [1, 2, 3],
    relatedDocs: ["/docs/quick-start", "/docs/commands/run"],
  },
  {
    number: 5,
    slug: "5-dashboard",
    title: "Explore the dashboard",
    description: "Navigate the azd-app dashboard and understand its features",
    estimatedTime: 5,
    icon: "📊",
    prerequisites: [4],
    relatedDocs: ["/docs/dashboard"],
  },
  {
    number: 6,
    slug: "6-logs",
    title: "View and filter logs",
    description: "Learn to view, filter, and search logs for debugging",
    estimatedTime: 5,
    icon: "📋",
    prerequisites: [4, 5],
    relatedDocs: ["/docs/commands/logs"],
  },
  {
    number: 7,
    slug: "7-health",
    title: "Monitor service health",
    description: "Monitor service health and troubleshoot issues",
    estimatedTime: 5,
    icon: "💓",
    prerequisites: [4, 5],
    relatedDocs: ["/docs/commands/health"],
  },
  {
    number: 8,
    slug: "8-mcp",
    title: "MCP server integration",
    description: "Connect GitHub Copilot to azd-app using the MCP server",
    estimatedTime: 10,
    icon: "🤖",
    prerequisites: [4],
    relatedDocs: ["/docs/mcp", "/docs/ai-features"],
  },
];

/** Get total estimated time for the entire tour */
export function getTotalEstimatedTime(): number {
  return tourSteps.reduce((sum, step) => sum + step.estimatedTime, 0);
}

/** Get a step by its slug */
export function getStepBySlug(slug: string): TourStep | undefined {
  return tourSteps.find(s => s.slug === slug);
}

/** Get a step by its number */
export function getStepByNumber(num: number): TourStep | undefined {
  return tourSteps.find(s => s.number === num);
}

/** Get the previous step */
export function getPreviousStep(currentNumber: number): TourStep | undefined {
  return tourSteps.find(s => s.number === currentNumber - 1);
}

/** Get the next step */
export function getNextStep(currentNumber: number): TourStep | undefined {
  return tourSteps.find(s => s.number === currentNumber + 1);
}

/** localStorage schema for tour progress */
export interface TourProgress {
  /** Version for migration */
  version: 1;
  /** Completed step numbers */
  completedSteps: number[];
  /** Last active step */
  lastActiveStep: number;
  /** Timestamp of last update */
  lastUpdated: string;
  /** Expanded "Learn More" sections */
  expandedSections: string[];
}

export const STORAGE_KEY = 'azd-app:tour:progress';

export const defaultProgress: TourProgress = {
  version: 1,
  completedSteps: [],
  lastActiveStep: 1,
  lastUpdated: new Date().toISOString(),
  expandedSections: [],
};
