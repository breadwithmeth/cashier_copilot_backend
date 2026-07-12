'use client'

import { FormEvent, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, RefreshCw, Search } from 'lucide-react'
import { api } from '@/lib/api'
import Layout from '@/components/Layout'
import { useParams } from 'next/navigation'

const titles: Record<string, string> = {
  cameras: 'Камеры',
  stores: 'Магазины',
  workplaces: 'Рабочие места',
  streams: 'Потоки',
  'product-scans': 'Сканы товаров',
  receipts: 'Чеки',
  'receipt-items': 'Позиции чеков',
  'sale-sessions': 'Сессии продаж',
  transcripts: 'Транскрипты',
  observations: 'Видео-наблюдения',
  'service-checks': 'Проверки обслуживания',
  'violation-types': 'Типы нарушений',
  'integration-errors': 'Ошибки интеграций',
  receivings: 'Приемка',
  workers: 'Воркеры',
  rules: 'Правила',
  models: 'Модели',
  users: 'Пользователи'
}

const columns: Record<string, string[]> = {
  stores: ['id', 'name', 'code', 'status', 'timezone'],
  workplaces: ['id', 'name', 'external_id', 'store_id', 'status'],
  cameras: ['id', 'name', 'code', 'workplace_id', 'status'],
  streams: ['id', 'camera_id', 'stream_type', 'status', 'stream_url'],
  receipts: ['id', 'external_receipt_id', 'workplace_id', 'payment_method', 'receipt_total', 'occurred_at'],
  'product-scans': ['id', 'barcode', 'product_name', 'workplace_id', 'external_receipt_id', 'occurred_at'],
  'sale-sessions': ['id', 'workplace_id', 'receipt_id', 'customer_present', 'status', 'started_at'],
  transcripts: ['id', 'event_id', 'receipt_id', 'speaker', 'text', 'started_at'],
  'integration-errors': ['id', 'source_system', 'entity_type', 'error_code', 'status', 'occurred_at'],
  workers: ['id', 'name', 'status', 'last_heartbeat_at'],
  users: ['id', 'email', 'full_name', 'role', 'is_active']
}

const createFields: Record<string, { key: string; label: string; type?: string; placeholder?: string }[]> = {
  stores: [
    { key: 'name', label: 'Название' },
    { key: 'code', label: 'Код' },
    { key: 'timezone', label: 'Таймзона', placeholder: 'Asia/Almaty' }
  ],
  workplaces: [
    { key: 'store_id', label: 'ID магазина' },
    { key: 'name', label: 'Название' },
    { key: 'external_id', label: 'ID в 1С' }
  ],
  cameras: [
    { key: 'workplace_id', label: 'ID рабочего места' },
    { key: 'name', label: 'Название' },
    { key: 'code', label: 'Код' }
  ],
  streams: [
    { key: 'camera_id', label: 'ID камеры' },
    { key: 'stream_type', label: 'Тип', placeholder: 'VIDEO_RTSP' },
    { key: 'stream_url', label: 'URL потока' }
  ]
}

function valueOf(row: any, key: string) {
  const value = row?.[key]
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? 'Да' : 'Нет'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function formatCell(row: any, key: string) {
  const raw = valueOf(row, key)
  if (key.endsWith('_at') && raw !== '-') return new Date(raw).toLocaleString('ru-RU')
  if (key === 'text' && raw.length > 140) return `${raw.slice(0, 140)}...`
  if (key === 'stream_url' && raw.length > 80) return `${raw.slice(0, 80)}...`
  return raw
}

export default function ResourcePage() {
  const { resource } = useParams()
  const name = String(resource)
  const title = titles[name] || name
  const [search, setSearch] = useState('')
  const [form, setForm] = useState<Record<string, string>>({})
  const queryClient = useQueryClient()

  const { data = [], isLoading, refetch } = useQuery({
    queryKey: ['resource', name],
    queryFn: () => api.get(`/resources/${name}`).then(r => r.data)
  })

  const fields = createFields[name] ?? []
  const selectedColumns = columns[name] ?? Object.keys(data[0] ?? {}).slice(0, 7)

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return data
    return data.filter((row: any) => selectedColumns.some(key => valueOf(row, key).toLowerCase().includes(term)))
  }, [data, search, selectedColumns])

  const createMutation = useMutation({
    mutationFn: () => api.post(`/resources/${name}`, form),
    onSuccess: () => {
      setForm({})
      queryClient.invalidateQueries({ queryKey: ['resource', name] })
    }
  })

  function submit(e: FormEvent) {
    e.preventDefault()
    createMutation.mutate()
  }

  return (
    <Layout>
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 className="text-3xl font-semibold">{title}</h1>
          <p className="muted mt-2">{isLoading ? 'Загрузка...' : `${filtered.length} из ${data.length} записей`}</p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <label className="flex min-w-[260px] items-center gap-2 rounded-lg border border-slate-800 bg-slate-900 px-3">
            <Search size={16} className="text-slate-500" />
            <input
              className="w-full border-0 bg-transparent px-0 outline-none"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Поиск"
            />
          </label>
          <button className="inline-flex items-center justify-center gap-2" onClick={() => refetch()}>
            <RefreshCw size={16} />
            Обновить
          </button>
        </div>
      </div>

      {fields.length > 0 && (
        <form className="card mt-6" onSubmit={submit}>
          <div className="grid gap-3 md:grid-cols-[repeat(3,minmax(0,1fr))_auto]">
            {fields.map(field => (
              <label key={field.key} className="grid gap-1 text-sm text-slate-300">
                {field.label}
                <input
                  value={form[field.key] ?? ''}
                  onChange={e => setForm(x => ({ ...x, [field.key]: e.target.value }))}
                  placeholder={field.placeholder}
                  required
                />
              </label>
            ))}
            <button className="mt-auto inline-flex items-center justify-center gap-2" disabled={createMutation.isPending}>
              <Plus size={16} />
              Создать
            </button>
          </div>
          {createMutation.isError && <p className="mt-3 text-sm text-red-300">Не удалось создать запись</p>}
        </form>
      )}

      <div className="card mt-6 overflow-x-auto p-0">
        <table className="w-full min-w-[860px] text-left text-sm">
          <thead className="border-b border-slate-800 text-xs uppercase text-slate-500">
            <tr>
              {selectedColumns.map(key => <th key={key} className="px-4 py-3 font-medium">{key}</th>)}
            </tr>
          </thead>
          <tbody>
            {filtered.map((row: any) => (
              <tr key={row.id} className="border-b border-slate-900 last:border-0">
                {selectedColumns.map(key => (
                  <td key={key} className="max-w-[360px] px-4 py-3 align-top text-slate-200">
                    <span className={key === 'status' && row[key] === 'OPEN' ? 'text-amber-300' : ''}>{formatCell(row, key)}</span>
                  </td>
                ))}
              </tr>
            ))}
            {!filtered.length && (
              <tr>
                <td className="px-4 py-10 text-center text-slate-500" colSpan={selectedColumns.length}>Нет записей</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Layout>
  )
}
