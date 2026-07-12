'use client'

import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import Layout from '@/components/Layout'
import { useParams } from 'next/navigation'

export default function EventDetailPage() {
  const { id } = useParams()
  const { data: e } = useQuery({
    queryKey: ['event', id],
    queryFn: () => api.get(`/events/${id}`).then(r => r.data)
  })

  if (!e) return null

  return (
    <Layout>
      <h1>{e.title}</h1>
      <div className="card mt-6">
        <p>{e.description}</p>
        {e.event_evidence?.map((x: any) => 
          x.evidence_type === 'IMAGE' && (
            <img 
              key={x.id}
              className="max-w-3xl mt-4" 
              src={x.public_url ?? x.file_path} 
              alt="Evidence"
            />
          )
        )}
      </div>
    </Layout>
  )
}
