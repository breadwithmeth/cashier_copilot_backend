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
      <h1>События</h1>
      <div className="card mt-6 overflow-auto">
        <table className="w-full text-left">
          <thead>
            <tr>
              <th>Время</th>
              <th>Тип</th>
              <th>Камера</th>
              <th>Статус</th>
            </tr>
          </thead>
          <tbody>
            {data?.data.map((e: any) => (
              <tr key={e.id} className="border-t border-slate-800">
                <td>
                  <Link href={`/events/${e.id}`}>
                    {new Date(e.started_at).toLocaleString('ru')}
                  </Link>
                </td>
                <td>{e.event_types.name}</td>
                <td>{e.cameras.name}</td>
                <td>{e.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Layout>
  )
}
