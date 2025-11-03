import { Table, TableHeader, TableBody, TableHead, TableRow } from '@/components/ui/table'
import { ServiceTableRow } from '@/components/ServiceTableRow'
import type { Service } from '@/types'

interface ServiceTableProps {
  services: Service[]
  onViewLogs?: (serviceName: string) => void
}

export function ServiceTable({ services, onViewLogs }: ServiceTableProps) {
  return (
    <div className="bg-[#1a1a1a] rounded-lg overflow-hidden border border-white/10">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent border-b border-white/10">
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
