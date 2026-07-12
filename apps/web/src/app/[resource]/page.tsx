'use client'

import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import Layout from '@/components/Layout'
import { useParams } from 'next/navigation'

const titles: Record<string, string> = {
  cameras: 'Камеры',
  stores: 'Магазины',
  workplaces: 'Рабочие места',
  workers: 'Воркеры',
  rules: 'Правила',
  models: 'Модели',
  users: 'Пользователи',
  reports: 'Отчёты'
}

export default function ResourcePage() {
  const { resource } = useParams()
  const title = titles[resource as string] || resource

  const { data } = useQuery({
    queryKey: [resource],
    queryFn: () => api.get(`/resources/${resource}`).then(r => r.data)
  })

  return (
    <Layout>
      <h1>{title}</h1>
      <div className="grid md:grid-cols-3 gap-4 mt-6">
        {data?.map((x: any) => (
          <div className="card" key={x.id}>
            <b>{x.name ?? x.full_name ?? x.code ?? x.id}</b>
            <p className="muted">{x.status ?? x.role ?? x.code}</p>
          </div>
        ))}
      </div>
    </Layout>
  )
}
