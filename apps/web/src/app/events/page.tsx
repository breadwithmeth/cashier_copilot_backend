'use client'

import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import Layout from '@/components/Layout'
import Link from 'next/link'

export default function EventsPage() {
  const { data } = useQuery({
    queryKey: ['events'],
    queryFn: () => api.get('/events').then(r => r.data)
  })

  return (
    <Layout>
      <h1 className="text-3xl font-semibold">События</h1>
      <div className="card mt-6 overflow-auto">
        <table className="w-full min-w-[900px] text-left text-sm">
          <thead className="text-xs uppercase text-slate-500">
            <tr>
              <th className="py-2">Время</th>
              <th>Тип</th>
              <th>Камера</th>
              <th>Чек</th>
              <th>Серьезность</th>
              <th>Статус</th>
            </tr>
          </thead>
          <tbody>
            {data?.data.map((e: any) => (
              <tr key={e.id} className="border-t border-slate-800">
                <td className="py-3">
                  <Link className="text-cyan-300" href={`/events/${e.id}`}>
                    {new Date(e.started_at).toLocaleString('ru')}
                  </Link>
                </td>
                <td>{e.violation_types?.name ?? e.event_types?.name ?? '-'}</td>
                <td>{e.cameras?.name ?? '-'}</td>
                <td>{e.receipts?.external_receipt_id ?? e.external_receipt_id ?? '-'}</td>
                <td>{e.severity}</td>
                <td>{e.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Layout>
  )
}
