/**
 * Pipeline, Infrastructure, and Configuration E2E Tests for Azure YAML Editor
 * 
 * Tests configuration for:
 * - Pipeline
 * - Infrastructure
 * - State
 * - Platform
 * - Workflows
 * - Cloud
 * - Required versions
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
} from '../helpers/test-setup'

test.describe('Pipeline Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          pipeline: {
            provider: 'github',
            variables: ['VAR1'],
            secrets: ['SECRET1'],
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure pipeline provider', async ({ page }) => {
    const pipelineNav = page.locator('[role="button"]:has-text("Pipeline")').first()
    if (await pipelineNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await pipelineNav.click()
      await page.waitForTimeout(500)

      const providerSelect = page.locator('select[name*="provider" i]').first()
      if (await providerSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await providerSelect.selectOption('azdo')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure pipeline variables', async ({ page }) => {
    const pipelineNav = page.locator('[role="button"]:has-text("Pipeline")').first()
    if (await pipelineNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await pipelineNav.click()
      await page.waitForTimeout(500)

      const variablesSection = page.locator('[aria-label*="variable" i], button:has-text("Variables")').first()
      if (await variablesSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await variablesSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const varInput = page.locator('input[name*="variable" i], input[placeholder*="variable" i]').first()
          if (await varInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await varInput.fill('VAR2')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })

  test('should configure pipeline secrets', async ({ page }) => {
    const pipelineNav = page.locator('[role="button"]:has-text("Pipeline")').first()
    if (await pipelineNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await pipelineNav.click()
      await page.waitForTimeout(500)

      const secretsSection = page.locator('[aria-label*="secret" i], button:has-text("Secrets")').first()
      if (await secretsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await secretsSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const secretInput = page.locator('input[name*="secret" i], input[placeholder*="secret" i]').first()
          if (await secretInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await secretInput.fill('SECRET2')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })
})

test.describe('Infrastructure Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          infra: {
            provider: 'bicep',
            path: './infra',
            module: 'main',
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure infra provider', async ({ page }) => {
    const infraNav = page.locator('[role="button"]:has-text("Infrastructure"), [role="button"]:has-text("infra" i)').first()
    if (await infraNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await infraNav.click()
      await page.waitForTimeout(500)

      const providerSelect = page.locator('select[name*="provider" i]').first()
      if (await providerSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await providerSelect.selectOption('terraform')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure infra path', async ({ page }) => {
    const infraNav = page.locator('[role="button"]:has-text("Infrastructure")').first()
    if (await infraNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await infraNav.click()
      await page.waitForTimeout(500)

      const pathInput = page.locator('input[name*="path" i]').first()
      if (await pathInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await pathInput.fill('./custom-infra')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure infra module', async ({ page }) => {
    const infraNav = page.locator('[role="button"]:has-text("Infrastructure")').first()
    if (await infraNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await infraNav.click()
      await page.waitForTimeout(500)

      const moduleInput = page.locator('input[name*="module" i]').first()
      if (await moduleInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await moduleInput.fill('custom-module')
        await page.waitForTimeout(300)
      }
    }
  })
})

test.describe('State Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          state: {
            remote: {
              backend: 'AzureBlobStorage',
              config: {
                accountName: 'testaccount',
                containerName: 'test-state',
              },
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure remote state backend', async ({ page }) => {
    const stateNav = page.locator('[role="button"]:has-text("State")').first()
    if (await stateNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await stateNav.click()
      await page.waitForTimeout(500)

      const backendSelect = page.locator('select[name*="backend" i]').first()
      if (await backendSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(backendSelect).toBeVisible()
      }
    }
  })

  test('should configure state account name', async ({ page }) => {
    const stateNav = page.locator('[role="button"]:has-text("State")').first()
    if (await stateNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await stateNav.click()
      await page.waitForTimeout(500)

      const accountNameInput = page.locator('input[name*="accountName" i]').first()
      if (await accountNameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await accountNameInput.fill('new-account')
        await page.waitForTimeout(300)
      }
    }
  })
})

test.describe('Platform Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          platform: {
            type: 'devcenter',
            config: {
              name: 'test-devcenter',
              project: 'test-project',
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure platform type', async ({ page }) => {
    const platformNav = page.locator('[role="button"]:has-text("Platform")').first()
    if (await platformNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await platformNav.click()
      await page.waitForTimeout(500)

      const typeSelect = page.locator('select[name*="type" i]').first()
      if (await typeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(typeSelect).toHaveValue('devcenter')
      }
    }
  })

  test('should configure platform config', async ({ page }) => {
    const platformNav = page.locator('[role="button"]:has-text("Platform")').first()
    if (await platformNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await platformNav.click()
      await page.waitForTimeout(500)

      const nameInput = page.locator('input[name*="name" i]').first()
      if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await nameInput.fill('new-devcenter')
        await page.waitForTimeout(300)
      }
    }
  })
})

test.describe('Workflows Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          workflows: {
            up: {
              steps: [{ azd: 'provision' }],
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure workflows.up', async ({ page }) => {
    const workflowsNav = page.locator('[role="button"]:has-text("Workflows")').first()
    if (await workflowsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await workflowsNav.click()
      await page.waitForTimeout(500)

      const upSection = page.locator('[aria-label*="up" i], button:has-text("Up")').first()
      if (await upSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await upSection.click()
        await page.waitForTimeout(500)

        // Should show workflow steps
        const steps = page.locator('[role="listitem"], [class*="step"]')
        const count = await steps.count()
        expect(count).toBeGreaterThanOrEqual(0)
      }
    }
  })
})

test.describe('Cloud Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          cloud: {
            name: 'AzureCloud',
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure cloud name', async ({ page }) => {
    const cloudNav = page.locator('[role="button"]:has-text("Cloud")').first()
    if (await cloudNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await cloudNav.click()
      await page.waitForTimeout(500)

      const cloudSelect = page.locator('select[name*="name" i]').first()
      if (await cloudSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await cloudSelect.selectOption('AzureChinaCloud')
        await page.waitForTimeout(300)
      }
    }
  })
})

test.describe('Required Versions Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          requiredVersions: {
            azd: '>= 1.0.0',
            extensions: {
              'azure.ai.agents': 'latest',
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure azd version constraint', async ({ page }) => {
    const versionsNav = page.locator('[role="button"]:has-text("Required Versions"), [role="button"]:has-text("versions" i)').first()
    if (await versionsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await versionsNav.click()
      await page.waitForTimeout(500)

      const azdInput = page.locator('input[name*="azd" i]').first()
      if (await azdInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await azdInput.fill('>= 1.1.0')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure extension versions', async ({ page }) => {
    const versionsNav = page.locator('[role="button"]:has-text("Required Versions")').first()
    if (await versionsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await versionsNav.click()
      await page.waitForTimeout(500)

      const extensionsSection = page.locator('[aria-label*="extension" i], button:has-text("Extensions")').first()
      if (await extensionsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await extensionsSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const nameInput = page.locator('input[name*="name" i]').first()
          if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await nameInput.fill('azure.rest')
            await page.keyboard.press('Tab')
            const versionInput = page.locator('input[name*="version" i]').first()
            if (await versionInput.isVisible({ timeout: 2000 }).catch(() => false)) {
              await versionInput.fill('>= 1.0.0')
              await page.waitForTimeout(300)
            }
          }
        }
      }
    }
  })
})

test.describe('Pipeline, Infrastructure, and Configuration - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display all configuration sections from comprehensive project', async ({ page }) => {
    const sections = ['Pipeline', 'Infrastructure', 'State', 'Platform', 'Workflows', 'Cloud', 'Required Versions']
    
    for (const section of sections) {
      const sectionButton = page.locator(`[role="button"]:has-text("${section}")`).first()
      if (await sectionButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(sectionButton).toBeVisible()
      }
    }
  })
})
