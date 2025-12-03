# Alternate Dashboard Design

## Overview

The dashboard currently follows a design heavily influenced by the Aspire Dashboard aesthetic. This feature introduces an alternate, unrestricted design mode that allows for a completely fresh visual approach while maintaining the same functionality.

## Requirements

### Design Mode Selection

- Users can switch between "Classic" (current Aspire-inspired) and "Modern" (new unrestricted design) modes
- The design mode is controlled via URL parameter: `?design=classic` or `?design=modern`
- Default mode is "Classic" to maintain backward compatibility
- Design preference persists in localStorage after initial selection
- URL parameter takes precedence over localStorage

### Classic Mode (Existing)

- Maintains current Aspire Dashboard-inspired design
- No visual changes to existing components
- Accessible via `?design=classic` or no parameter

### Modern Mode (New)

- Completely fresh visual design with no constraints from previous art
- Maintains functional parity with Classic mode
- All existing features must work identically
- Design should feel innovative and contemporary

#### Modern Mode Design Principles

- Clean, minimal aesthetic
- Bold typography choices
- Generous whitespace
- Subtle, purposeful animations
- Distinctive color palette (different from Aspire purple)
- Card-based layouts with depth and shadows
- Fluid, responsive design
- Focus on content hierarchy and scannability
- Accessible by default (WCAG 2.1 AA)

### Functional Requirements

- Both modes share the same data layer and business logic
- Navigation structure remains consistent across modes
- All views must be implemented: Resources, Console, Environment, Metrics
- Keyboard shortcuts work identically in both modes
- Theme toggle (light/dark) works in both modes
- Real-time health monitoring displays in both modes
- Service detail panel functions in both modes

### Technical Implementation

- Design mode context provider at app root
- Conditional component rendering based on design mode
- Shared hooks and data fetching logic
- Separate component directories for each design mode
- CSS variables for design-specific theming
- URL parameter parsing in main entry point

## Success Criteria

- User can switch between Classic and Modern designs via URL parameter
- Modern design is visually distinct and does not reference Aspire aesthetics
- All dashboard functionality works in both modes
- Design preference persists across sessions
- No regression in Classic mode appearance or behavior
- Modern mode meets accessibility standards (WCAG 2.1 AA)
- Performance is not degraded in either mode
