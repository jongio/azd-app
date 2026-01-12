/**
 * Application Service Tab
 * Form for adding custom application services (code projects)
 */

import * as React from 'react'
import { useForm } from 'react-hook-form'
import { Folder } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ServiceFormData } from '@/lib/editor/wellknown-types'

export interface ApplicationServiceTabProps {
  /** Callback when form is submitted */
  onSubmit: (data: ServiceFormData) => void
  /** Initial form values */
  defaultValues?: Partial<ServiceFormData>
  /** Whether form is submitting */
  isSubmitting?: boolean
}

const HOST_TYPES = [
  { value: 'containerapp', label: 'Azure Container Apps' },
  { value: 'appservice', label: 'Azure App Service' },
  { value: 'function', label: 'Azure Functions' },
  { value: 'springapp', label: 'Azure Spring Apps' },
  { value: 'staticwebapp', label: 'Azure Static Web Apps' },
  { value: 'aks', label: 'Azure Kubernetes Service' },
]

const LANGUAGES = [
  { value: '', label: 'Auto-detect' },
  { value: 'node', label: 'Node.js' },
  { value: 'python', label: 'Python' },
  { value: 'dotnet', label: '.NET' },
  { value: 'java', label: 'Java' },
  { value: 'go', label: 'Go' },
  { value: 'rust', label: 'Rust' },
  { value: 'php', label: 'PHP' },
  { value: 'ruby', label: 'Ruby' },
]

/**
 * Application Service Tab Component
 */
export function ApplicationServiceTab({
  onSubmit,
  defaultValues = {},
  isSubmitting = false,
}: ApplicationServiceTabProps) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<ServiceFormData>({
    defaultValues: {
      host: 'containerapp',
      language: '',
      ports: [],
      environment: {},
      ...defaultValues,
    },
  })

  const hostType = watch('host')

  const handleFormSubmit = (data: ServiceFormData) => {
    // Clean up empty values
    const cleanData: ServiceFormData = {
      name: data.name,
      host: data.host,
      project: data.project,
    }

    if (data.language) {
      cleanData.language = data.language
    }

    if (data.ports && data.ports.length > 0) {
      cleanData.ports = data.ports
    }

    onSubmit(cleanData)
  }

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      {/* Service Name */}
      <div>
        <label
          htmlFor="app-service-name"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Service Name <span className="text-red-500">*</span>
        </label>
        <input
          id="app-service-name"
          type="text"
          {...register('name', {
            required: 'Service name is required',
            pattern: {
              value: /^[a-z0-9-]+$/,
              message: 'Service name must contain only lowercase letters, numbers, and hyphens',
            },
          })}
          className={cn(
            'w-full px-3 py-2 rounded-md border text-sm',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            'bg-white dark:bg-slate-800',
            'text-slate-900 dark:text-slate-100',
            errors.name
              ? 'border-red-500'
              : 'border-slate-300 dark:border-slate-600'
          )}
          placeholder="my-api"
          aria-invalid={errors.name ? 'true' : 'false'}
          aria-describedby={errors.name ? 'app-service-name-error' : undefined}
        />
        {errors.name && (
          <p id="app-service-name-error" className="mt-1 text-xs text-red-500">
            {errors.name.message}
          </p>
        )}
      </div>

      {/* Host Type */}
      <div>
        <label
          htmlFor="app-service-host"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Host Type <span className="text-red-500">*</span>
        </label>
        <select
          id="app-service-host"
          {...register('host', { required: 'Host type is required' })}
          className={cn(
            'w-full px-3 py-2 rounded-md border text-sm',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            'bg-white dark:bg-slate-800',
            'text-slate-900 dark:text-slate-100',
            errors.host
              ? 'border-red-500'
              : 'border-slate-300 dark:border-slate-600'
          )}
        >
          {HOST_TYPES.map((type) => (
            <option key={type.value} value={type.value}>
              {type.label}
            </option>
          ))}
        </select>
        {errors.host && (
          <p className="mt-1 text-xs text-red-500">{errors.host.message}</p>
        )}
      </div>

      {/* Project Path */}
      <div>
        <label
          htmlFor="app-service-project"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Project Path <span className="text-red-500">*</span>
        </label>
        <div className="relative">
          <input
            id="app-service-project"
            type="text"
            {...register('project', { required: 'Project path is required' })}
            className={cn(
              'w-full pl-10 pr-3 py-2 rounded-md border text-sm',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'bg-white dark:bg-slate-800',
              'text-slate-900 dark:text-slate-100',
              errors.project
                ? 'border-red-500'
                : 'border-slate-300 dark:border-slate-600'
            )}
            placeholder="./src/api"
            aria-invalid={errors.project ? 'true' : 'false'}
            aria-describedby={errors.project ? 'app-service-project-error' : undefined}
          />
          <Folder className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        </div>
        {errors.project && (
          <p id="app-service-project-error" className="mt-1 text-xs text-red-500">
            {errors.project.message}
          </p>
        )}
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Relative path to your application code
        </p>
      </div>

      {/* Language */}
      <div>
        <label
          htmlFor="app-service-language"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Language
        </label>
        <select
          id="app-service-language"
          {...register('language')}
          className={cn(
            'w-full px-3 py-2 rounded-md border text-sm',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            'bg-white dark:bg-slate-800',
            'text-slate-900 dark:text-slate-100',
            'border-slate-300 dark:border-slate-600'
          )}
        >
          {LANGUAGES.map((lang) => (
            <option key={lang.value} value={lang.value}>
              {lang.label}
            </option>
          ))}
        </select>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Leave as "Auto-detect" to automatically detect the language from your project
        </p>
      </div>

      {/* Host-specific hints */}
      {hostType === 'function' && (
        <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 p-3">
          <p className="text-sm text-blue-800 dark:text-blue-300">
            💡 <strong>Azure Functions:</strong> Make sure your project has a host.json file and function definitions.
          </p>
        </div>
      )}

      {hostType === 'staticwebapp' && (
        <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 p-3">
          <p className="text-sm text-blue-800 dark:text-blue-300">
            💡 <strong>Static Web Apps:</strong> Your project should contain static assets (HTML, CSS, JS) or a build output directory.
          </p>
        </div>
      )}

      {/* Submit Button */}
      <div className="flex justify-end pt-4 border-t border-slate-200 dark:border-slate-700">
        <button
          type="submit"
          disabled={isSubmitting}
          className={cn(
            'px-6 py-2.5 rounded-lg text-sm font-semibold shadow-sm',
            'bg-cyan-600 text-white hover:bg-cyan-700',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'transition-colors duration-150'
          )}
        >
          {isSubmitting ? 'Adding Service...' : 'Add Service'}
        </button>
      </div>
    </form>
  )
}
