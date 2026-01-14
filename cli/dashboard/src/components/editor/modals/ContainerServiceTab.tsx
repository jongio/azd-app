/**
 * Container Service Tab
 * Form for adding custom container services (Docker images)
 */

import * as React from 'react'
import { useForm } from 'react-hook-form'
import { Container } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ServiceFormData } from '@/lib/editor/wellknown-types'

export interface ContainerServiceTabProps {
  /** Callback when form is submitted */
  onSubmit: (data: ServiceFormData) => void
  /** Initial form values */
  defaultValues?: Partial<ServiceFormData>
  /** Whether form is submitting */
  isSubmitting?: boolean
}

const HOST_TYPES = [
  { value: 'containerapp', label: 'Azure Container Apps' },
  { value: 'aks', label: 'Azure Kubernetes Service' },
]

/**
 * Container Service Tab Component
 */
export function ContainerServiceTab({
  onSubmit,
  defaultValues = {},
  isSubmitting = false,
}: ContainerServiceTabProps) {
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<ServiceFormData>({
    defaultValues: {
      host: 'containerapp',
      ports: [],
      environment: {},
      ...defaultValues,
    },
  })

  const [portsInput, setPortsInput] = React.useState('')

  const handleFormSubmit = (data: ServiceFormData) => {
    // Parse ports from input
    const ports = portsInput
      .split(',')
      .map(p => p.trim())
      .filter(p => p.length > 0)

    // Clean up empty values
    const cleanData: ServiceFormData = {
      name: data.name,
      host: data.host,
      image: data.image,
    }

    if (ports.length > 0) {
      cleanData.ports = ports
    }

    onSubmit(cleanData)
  }

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      {/* Service Name */}
      <div>
        <label
          htmlFor="container-service-name"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Service Name <span className="text-red-500">*</span>
        </label>
        <input
          id="container-service-name"
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
          placeholder="my-service"
          aria-invalid={errors.name ? 'true' : 'false'}
          aria-describedby={errors.name ? 'container-service-name-error' : undefined}
        />
        {errors.name && (
          <p id="container-service-name-error" className="mt-1 text-xs text-red-500">
            {errors.name.message}
          </p>
        )}
      </div>

      {/* Host Type */}
      <div>
        <label
          htmlFor="container-service-host"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Host Type <span className="text-red-500">*</span>
        </label>
        <select
          id="container-service-host"
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

      {/* Docker Image */}
      <div>
        <label
          htmlFor="container-service-image"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Docker Image <span className="text-red-500">*</span>
        </label>
        <div className="relative">
          <input
            id="container-service-image"
            type="text"
            {...register('image', {
              required: 'Docker image is required',
              pattern: {
                value: /^[a-z0-9.\-_/:]+$/i,
                message: 'Invalid Docker image format',
              },
            })}
            className={cn(
              'w-full pl-10 pr-3 py-2 rounded-md border text-sm',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'bg-white dark:bg-slate-800',
              'text-slate-900 dark:text-slate-100',
              errors.image
                ? 'border-red-500'
                : 'border-slate-300 dark:border-slate-600'
            )}
            placeholder="nginx:latest"
            aria-invalid={errors.image ? 'true' : 'false'}
            aria-describedby={errors.image ? 'container-service-image-error' : undefined}
          />
          <Container className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        </div>
        {errors.image && (
          <p id="container-service-image-error" className="mt-1 text-xs text-red-500">
            {errors.image.message}
          </p>
        )}
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Full image name with optional tag (e.g., nginx:latest, mcr.microsoft.com/redis:7)
        </p>
      </div>

      {/* Port Mappings */}
      <div>
        <label
          htmlFor="container-service-ports"
          className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
        >
          Port Mappings
        </label>
        <input
          id="container-service-ports"
          type="text"
          value={portsInput}
          onChange={(e) => setPortsInput(e.target.value)}
          className={cn(
            'w-full px-3 py-2 rounded-md border text-sm',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            'bg-white dark:bg-slate-800',
            'text-slate-900 dark:text-slate-100',
            'border-slate-300 dark:border-slate-600'
          )}
          placeholder="8080:80, 8443:443"
        />
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Comma-separated port mappings (e.g., "8080:80, 8443:443" or just "80, 443")
        </p>
      </div>

      {/* Common Images Examples */}
      <div className="rounded-lg bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-4">
        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">
          Common Container Images
        </h4>
        <div className="space-y-2">
          <button
            type="button"
            onClick={() => {
              setValue('image', 'nginx:alpine')
              setPortsInput('80:80')
            }}
            className="block w-full text-left px-3 py-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-sm"
          >
            <span className="font-medium text-slate-900 dark:text-slate-100">Nginx</span>
            <span className="text-slate-500 dark:text-slate-400 ml-2">→ nginx:alpine</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setValue('image', 'redis:7-alpine')
              setPortsInput('6379:6379')
            }}
            className="block w-full text-left px-3 py-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-sm"
          >
            <span className="font-medium text-slate-900 dark:text-slate-100">Redis</span>
            <span className="text-slate-500 dark:text-slate-400 ml-2">→ redis:7-alpine</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setValue('image', 'postgres:16-alpine')
              setPortsInput('5432:5432')
            }}
            className="block w-full text-left px-3 py-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-sm"
          >
            <span className="font-medium text-slate-900 dark:text-slate-100">PostgreSQL</span>
            <span className="text-slate-500 dark:text-slate-400 ml-2">→ postgres:16-alpine</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setValue('image', 'mongo:7')
              setPortsInput('27017:27017')
            }}
            className="block w-full text-left px-3 py-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-sm"
          >
            <span className="font-medium text-slate-900 dark:text-slate-100">MongoDB</span>
            <span className="text-slate-500 dark:text-slate-400 ml-2">→ mongo:7</span>
          </button>
        </div>
      </div>

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
