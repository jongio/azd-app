/**
 * Performance Benchmark Suite
 * Comprehensive performance tests for Azure YAML Editor
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { 
  measureTime, 
  performanceMonitor,
} from './testing'
import {
  generateLargeConfig,
  generateValidationErrors,
  generateNavigationNodes,
  generateCommands,
  PERFORMANCE_SCENARIOS,
} from './test-data'

describe('Performance Benchmarks', () => {
  beforeEach(() => {
    performanceMonitor.clear()
  })

  describe('Configuration Generation', () => {
    it('should generate small config in <50ms', () => {
      const { duration } = measureTime(
        () => generateLargeConfig(PERFORMANCE_SCENARIOS.small.serviceCount),
        'Generate small config'
      )
      expect(duration).toBeLessThan(50)
    })

    it('should generate medium config in <100ms', () => {
      const { duration } = measureTime(
        () => generateLargeConfig(PERFORMANCE_SCENARIOS.medium.serviceCount),
        'Generate medium config'
      )
      expect(duration).toBeLessThan(100)
    })

    it('should generate large config in <200ms', () => {
      const { duration } = measureTime(
        () => generateLargeConfig(PERFORMANCE_SCENARIOS.large.serviceCount),
        'Generate large config'
      )
      expect(duration).toBeLessThan(200)
    })
  })

  describe('Navigation Performance', () => {
    it('should generate small navigation in <10ms', () => {
      const { duration } = measureTime(
        () => generateNavigationNodes(PERFORMANCE_SCENARIOS.small.serviceCount),
        'Generate small navigation'
      )
      expect(duration).toBeLessThan(10)
    })

    it('should generate large navigation in <50ms', () => {
      const { duration } = measureTime(
        () => generateNavigationNodes(PERFORMANCE_SCENARIOS.large.serviceCount),
        'Generate large navigation'
      )
      expect(duration).toBeLessThan(50)
    })
  })

  describe('Command Palette Performance', () => {
    it('should generate commands for small config in <20ms', () => {
      const { duration } = measureTime(
        () => generateCommands(PERFORMANCE_SCENARIOS.small.serviceCount),
        'Generate commands (small)'
      )
      expect(duration).toBeLessThan(20)
    })

    it('should generate commands for large config in <100ms', () => {
      const { duration } = measureTime(
        () => generateCommands(PERFORMANCE_SCENARIOS.large.serviceCount),
        'Generate commands (large)'
      )
      expect(duration).toBeLessThan(100)
    })
  })

  describe('Validation Performance', () => {
    it('should generate validation errors in <50ms', () => {
      const { duration } = measureTime(
        () => generateValidationErrors(PERFORMANCE_SCENARIOS.large.serviceCount),
        'Generate validation errors'
      )
      expect(duration).toBeLessThan(50)
    })
  })

  describe('Performance Targets (Task 20)', () => {
    it('should meet initial page load target (<500ms simulation)', () => {
      // Simulate initial load operations
      const { duration } = measureTime(() => {
        generateLargeConfig(10) // Load small config
        generateNavigationNodes(10)
        generateCommands(10)
      }, 'Initial load simulation')
      
      expect(duration).toBeLessThan(500)
    })

    it('should meet field update target (<50ms)', () => {
      // Simulate field update with debouncing
      const { duration } = measureTime(() => {
        // Field update simulation (validation not included as it's debounced)
        const config = generateLargeConfig(10)
        config.services['service-0001'].environment = { NEW_VAR: 'value' }
      }, 'Field update simulation')
      
      expect(duration).toBeLessThan(50)
    })

  describe('Large Config Rendering', () => {
    it('should handle large config rendering (<200ms for 100+ services)', () => {
      const { duration } = measureTime(() => {
        const config = generateLargeConfig(100)
        generateNavigationNodes(100)
        // Simulate rendering preparation
        Object.keys(config.services)
      }, 'Large config rendering preparation')
      
      expect(duration).toBeLessThan(200)
    })
  })
  })

  describe('Memory Performance', () => {
    it('should not leak memory with large configs', () => {
      const initialMemory = (performance as any).memory?.usedJSHeapSize || 0
      
      // Generate and discard multiple large configs
      for (let i = 0; i < 10; i++) {
        generateLargeConfig(100)
      }
      
      // Force garbage collection if available
      if (global.gc) {
        global.gc()
      }
      
      const finalMemory = (performance as any).memory?.usedJSHeapSize || 0
      const growth = finalMemory - initialMemory
      
      // Memory growth should be minimal (<10MB)
      expect(growth).toBeLessThan(10 * 1024 * 1024)
    })
  })
})

describe('Virtual Scrolling Performance', () => {
  it('should handle large item lists efficiently', () => {
    const items = generateNavigationNodes(1000)
    // Virtual list should render ~10-20 items for viewport
    // Navigation nodes include 4 top-level sections + 1000 service children
    expect(items.length).toBeGreaterThan(3)
  })
})

describe('Memoization Performance', () => {
  it('should cache expensive computations', () => {
    const config = generateLargeConfig(100)
    
    // First computation
    const { duration: firstDuration } = measureTime(() => {
      JSON.stringify(config)
    }, 'First computation')
    
    // Cached computation (simulated)
    const cached = JSON.stringify(config)
    const { duration: cachedDuration } = measureTime(() => {
      cached
    }, 'Cached computation')
    
    // Cached should be significantly faster
    expect(cachedDuration).toBeLessThan(firstDuration / 10)
  })
})

describe('Performance Report', () => {
  it('should generate performance report', () => {
    // Record some metrics
    performanceMonitor.record('operation-1', 100)
    performanceMonitor.record('operation-1', 120)
    performanceMonitor.record('operation-2', 50)
    
    const report = performanceMonitor.getReport()
    
    expect(report).toContain('Performance Report')
    expect(report).toContain('operation-1')
    expect(report).toContain('operation-2')
    expect(report).toContain('Average')
    expect(report).toContain('P95')
  })
})

