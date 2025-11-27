import { Table, TableHeader, TableBody, TableHead, TableRow } from '@/components/ui/table'
import { ServiceTableRow } from '@/components/ServiceTableRow'
import type { Service } from '@/types'

interface ServiceTableProps {
  services: Service[]
  onViewLogs?: (serviceName: string) => void
}

export function ServiceTable({ services, onViewLogs }: ServiceTableProps) {
  return (
    <div className="bg-card rounded-lg overflow-hidden border border-card-border">
      <div className="flex items-center justify-between p-4 border-b border-border">
        <h2 className="text-sm font-semibold text-foreground">Services ({services.length})</h2>
      </div>
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent border-b border-border">
            <TableHead className="w-[180px]">Name</TableHead>
            <TableHead className="w-[120px]">State</TableHead>
            <TableHead className="w-[140px]">Start time</TableHead>
            <TableHead className="min-w-[200px]">Source</TableHead>
            <TableHead className="min-w-[200px]">Local URL</TableHead>
            <TableHead className="min-w-[200px]">Azure URL</TableHead>
            <TableHead className="w-[100px] text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {services.map((service) => (
            <ServiceTableRow 
              key={service.name} 
              service={service}
              onViewLogs={onViewLogs}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
