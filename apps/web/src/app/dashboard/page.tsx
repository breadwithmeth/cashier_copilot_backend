'use client'

import Layout from '@/components/Layout'
import { api } from '@/lib/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Activity, AlertTriangle, Camera, CircleDot, Mic, Monitor, Plus, Radio, Store, UsersRound, Wifi, WifiOff } from 'lucide-react'
import Link from 'next/link'
import type { FormEvent } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

type DashboardSummary = {
  totals: {
    cameras: number
    online: number
    offline: number
    events24h: number
    critical24h: number
    stores: number
    workers: number
  }
  cameraStatuses: { status: string; count: number }[]
  eventSeverities: { severity: string; count: number }[]
  recentEvents: {
    id: string
    title: string
    severity: string
    status: string
    startedAt: string
    camera: { name: string; code: string } | null
    eventType: { name: string; code: string } | null
  }[]
  problemCameras: {
    id: string
    name: string
    code: string
    status: string
    processingEnabled: boolean
    lastFrameAt: string | null
    store: { name: string; code: string } | null
  }[]
  workers: {
    id: string
    name: string
    host: string
    version: string
    status: string
    lastHeartbeatAt: string | null
  }[]
}

type StoreRow = { id: string; name: string; code: string }
type WorkplaceRow = { id: string; name: string; external_id: string; store_id: string }
type CameraRow = { id: string; name: string; code: string; store_id: string; workplace_id: string }

const severityColors: Record<string, string> = {
  CRITICAL: '#ef4444',
  WARNING: '#f59e0b',
  INFO: '#06b6d4',
}

const statusColors: Record<string, string> = {
  ONLINE: '#22c55e',
  OFFLINE: '#64748b',
  ERROR: '#ef4444',
  BUSY: '#f59e0b',
  unknown: '#64748b',
}

function formatDate(value?: string | null) {
  if (!value) return 'нет данных'
  return new Intl.DateTimeFormat('ru', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function Badge({ value, tone }: { value: string; tone?: string }) {
  return (
    <span className="inline-flex h-7 items-center rounded-full border border-slate-700 px-2.5 text-xs text-slate-200">
      <span className="mr-2 h-2 w-2 rounded-full" style={{ backgroundColor: tone ?? '#64748b' }} />
      {value}
    </span>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="rounded-lg border border-dashed border-slate-800 p-6 text-center text-sm text-slate-400">{text}</div>
}

function field(form: HTMLFormElement, name: string) {
  return String(new FormData(form).get(name) ?? '').trim()
}

export default function DashboardPage() {
  const queryClient = useQueryClient()
  const { data, isLoading, isError, refetch, isFetching } = useQuery<DashboardSummary>({
    queryKey: ['dashboard-summary'],
    queryFn: () => api.get('/dashboard/summary').then(r => r.data),
    refetchInterval: 30000,
  })
  const { data: stores = [] } = useQuery<StoreRow[]>({
    queryKey: ['stores'],
    queryFn: () => api.get('/resources/stores').then(r => r.data),
  })
  const { data: workplaces = [] } = useQuery<WorkplaceRow[]>({
    queryKey: ['workplaces'],
    queryFn: () => api.get('/resources/workplaces').then(r => r.data),
  })
  const { data: cameras = [] } = useQuery<CameraRow[]>({
    queryKey: ['cameras'],
    queryFn: () => api.get('/resources/cameras').then(r => r.data),
  })
  const invalidateData = () => {
    queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] })
    queryClient.invalidateQueries({ queryKey: ['stores'] })
    queryClient.invalidateQueries({ queryKey: ['workplaces'] })
    queryClient.invalidateQueries({ queryKey: ['cameras'] })
  }
  const createStore = useMutation({
    mutationFn: (payload: Record<string, unknown>) => api.post('/resources/stores', payload),
    onSuccess: invalidateData,
  })
  const createCamera = useMutation({
    mutationFn: (payload: Record<string, unknown>) => api.post('/resources/cameras', payload),
    onSuccess: invalidateData,
  })
  const createWorkplace = useMutation({
    mutationFn: (payload: Record<string, unknown>) => api.post('/resources/workplaces', payload),
    onSuccess: invalidateData,
  })
  const createStream = useMutation({
    mutationFn: (payload: Record<string, unknown>) => api.post('/resources/streams', payload),
    onSuccess: invalidateData,
  })

  const totals = data?.totals
  const cameraChart = data?.cameraStatuses?.length
    ? data.cameraStatuses
    : [{ status: 'нет данных', count: 1 }]
  const severityChart = data?.eventSeverities ?? []

  return (
    <Layout>
      <div className="mb-6 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal">Главная панель</h1>
          <p className="mt-1 text-sm text-slate-400">Оперативное состояние камер, событий и analytics worker-ов.</p>
        </div>
        <button className="self-start md:self-auto" onClick={() => refetch()} disabled={isFetching}>
          {isFetching ? 'Обновление...' : 'Обновить'}
        </button>
      </div>

      {isError && (
        <div className="mb-6 rounded-lg border border-red-900 bg-red-950/40 p-4 text-sm text-red-100">
          Не удалось загрузить дашборд. Проверь API и авторизацию.
        </div>
      )}

      <section className="card mb-6">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">Быстрое добавление</h2>
            <p className="mt-1 text-sm text-slate-400">Заполни цепочку: магазин, рабочее место, камера, поток.</p>
          </div>
          <Plus className="h-5 w-5 text-cyan-300" />
        </div>
        <div className="grid gap-4 xl:grid-cols-5">
          <form
            className="rounded-lg border border-slate-800 p-4"
            onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault()
              const form = event.currentTarget
              createStore.mutate({
                name: field(form, 'name'),
                code: field(form, 'code'),
                city: field(form, 'city') || undefined,
                address: field(form, 'address') || undefined,
              }, { onSuccess: () => form.reset() })
            }}
          >
            <div className="mb-3 flex items-center gap-2 font-medium"><Store className="h-4 w-4 text-emerald-300" />Магазин</div>
            <div className="space-y-3">
              <input className="w-full" name="name" placeholder="Название" required />
              <input className="w-full" name="code" placeholder="Код, например store-1" required />
              <input className="w-full" name="city" placeholder="Город" />
              <input className="w-full" name="address" placeholder="Адрес" />
              <button className="w-full" disabled={createStore.isPending}>{createStore.isPending ? 'Сохранение...' : 'Добавить магазин'}</button>
            </div>
          </form>

          <form
            className="rounded-lg border border-slate-800 p-4"
            onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault()
              const form = event.currentTarget
              createWorkplace.mutate({
                store_id: field(form, 'store_id'),
                name: field(form, 'name'),
                external_id: field(form, 'external_id'),
                workplace_type: 'checkout',
                is_active: true,
              }, { onSuccess: () => form.reset() })
            }}
          >
            <div className="mb-3 flex items-center gap-2 font-medium"><Monitor className="h-4 w-4 text-sky-300" />Рабочее место</div>
            <div className="space-y-3">
              <select className="w-full" name="store_id" required defaultValue="">
                <option value="" disabled>Выбери магазин</option>
                {stores.map(s => <option key={s.id} value={s.id}>{s.name} · {s.code}</option>)}
              </select>
              <input className="w-full" name="name" placeholder="Название кассы" required />
              <input className="w-full" name="external_id" placeholder="ID из 1С, например pos-1" required />
              <button className="w-full" disabled={createWorkplace.isPending || !stores.length}>{createWorkplace.isPending ? 'Сохранение...' : 'Добавить место'}</button>
            </div>
          </form>

          <form
            className="rounded-lg border border-slate-800 p-4"
            onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault()
              const form = event.currentTarget
              createCamera.mutate({
                workplace_id: field(form, 'workplace_id'),
                name: field(form, 'name'),
                code: field(form, 'code'),
                location_description: field(form, 'location_description') || undefined,
                processing_enabled: true,
                is_active: true,
              }, { onSuccess: () => form.reset() })
            }}
          >
            <div className="mb-3 flex items-center gap-2 font-medium"><Camera className="h-4 w-4 text-cyan-300" />Камера</div>
            <div className="space-y-3">
              <select className="w-full" name="workplace_id" required defaultValue="">
                <option value="" disabled>Выбери рабочее место</option>
                {workplaces.map(w => <option key={w.id} value={w.id}>{w.name} · {w.external_id}</option>)}
              </select>
              <input className="w-full" name="name" placeholder="Название камеры" required />
              <input className="w-full" name="code" placeholder="Код, например checkout-1" required />
              <input className="w-full" name="location_description" placeholder="Расположение" />
              <button className="w-full" disabled={createCamera.isPending || !workplaces.length}>{createCamera.isPending ? 'Сохранение...' : 'Добавить камеру'}</button>
            </div>
          </form>

          <form
            className="rounded-lg border border-slate-800 p-4"
            onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault()
              const form = event.currentTarget
              createStream.mutate({
                camera_id: field(form, 'camera_id'),
                stream_type: 'RTSP_VIDEO',
                stream_url: field(form, 'stream_url'),
                transport: 'tcp',
                is_primary: true,
                is_enabled: true,
              }, { onSuccess: () => form.reset() })
            }}
          >
            <div className="mb-3 flex items-center gap-2 font-medium"><Radio className="h-4 w-4 text-violet-300" />Видео RTSP</div>
            <div className="space-y-3">
              <select className="w-full" name="camera_id" required defaultValue="">
                <option value="" disabled>Выбери камеру</option>
                {cameras.map(c => <option key={c.id} value={c.id}>{c.name} · {c.code}</option>)}
              </select>
              <input className="w-full" name="stream_url" placeholder="rtsp://user:pass@host:554/video" required />
              <button className="w-full" disabled={createStream.isPending || !cameras.length}>{createStream.isPending ? 'Сохранение...' : 'Добавить видео'}</button>
            </div>
          </form>

          <form
            className="rounded-lg border border-slate-800 p-4"
            onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault()
              const form = event.currentTarget
              createStream.mutate({
                camera_id: field(form, 'camera_id'),
                stream_type: 'RTSP_AUDIO',
                stream_url: field(form, 'stream_url'),
                transport: 'tcp',
                is_primary: false,
                is_enabled: true,
              }, { onSuccess: () => form.reset() })
            }}
          >
            <div className="mb-3 flex items-center gap-2 font-medium"><Mic className="h-4 w-4 text-amber-300" />Аудио RTSP</div>
            <div className="space-y-3">
              <select className="w-full" name="camera_id" required defaultValue="">
                <option value="" disabled>Выбери камеру</option>
                {cameras.map(c => <option key={c.id} value={c.id}>{c.name} · {c.code}</option>)}
              </select>
              <input className="w-full" name="stream_url" placeholder="rtsp://user:pass@host:554/audio" required />
              <button className="w-full" disabled={createStream.isPending || !cameras.length}>{createStream.isPending ? 'Сохранение...' : 'Добавить аудио'}</button>
            </div>
          </form>
        </div>
        {(createStore.isError || createWorkplace.isError || createCamera.isError || createStream.isError) && (
          <div className="mt-4 rounded-lg border border-red-900 bg-red-950/40 p-3 text-sm text-red-100">
            Не удалось сохранить. Проверь обязательные поля и права пользователя.
          </div>
        )}
      </section>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {[
          { label: 'Камеры', value: totals?.cameras, icon: Camera, detail: `${totals?.online ?? 0} онлайн`, color: 'text-cyan-300' },
          { label: 'События за 24ч', value: totals?.events24h, icon: Activity, detail: `${totals?.critical24h ?? 0} критических`, color: 'text-amber-300' },
          { label: 'Магазины', value: totals?.stores, icon: Store, detail: 'активная сеть', color: 'text-emerald-300' },
          { label: 'Воркеры', value: totals?.workers, icon: UsersRound, detail: 'узлы аналитики', color: 'text-violet-300' },
        ].map(item => (
          <div className="card min-h-32" key={item.label}>
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="text-sm text-slate-400">{item.label}</div>
                <div className="mt-3 text-3xl font-semibold">{isLoading ? '-' : item.value ?? 0}</div>
              </div>
              <item.icon className={`h-6 w-6 ${item.color}`} />
            </div>
            <div className="mt-4 text-sm text-slate-400">{item.detail}</div>
          </div>
        ))}
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-[1fr_1.2fr]">
        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Статусы камер</h2>
            <Badge value={`${totals?.offline ?? 0} офлайн`} tone={statusColors.OFFLINE} />
          </div>
          <div className="h-72">
            <ResponsiveContainer>
              <PieChart>
                <Pie data={cameraChart} dataKey="count" nameKey="status" innerRadius={58} outerRadius={92} paddingAngle={2}>
                  {cameraChart.map(x => (
                    <Cell key={x.status} fill={statusColors[x.status] ?? statusColors.unknown} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
          <div className="flex flex-wrap gap-2">
            {data?.cameraStatuses?.map(x => <Badge key={x.status} value={`${x.status}: ${x.count}`} tone={statusColors[x.status]} />)}
          </div>
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">События за 7 дней</h2>
            <Badge value="по severity" tone="#06b6d4" />
          </div>
          {severityChart.length ? (
            <div className="h-72">
              <ResponsiveContainer>
                <BarChart data={severityChart}>
                  <CartesianGrid stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="severity" stroke="#94a3b8" />
                  <YAxis allowDecimals={false} stroke="#94a3b8" />
                  <Tooltip cursor={{ fill: '#0f172a' }} />
                  <Bar dataKey="count" radius={[6, 6, 0, 0]}>
                    {severityChart.map(x => (
                      <Cell key={x.severity} fill={severityColors[x.severity] ?? '#64748b'} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <EmptyState text="За последние 7 дней событий нет" />
          )}
        </section>
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-2">
        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Последние события</h2>
            <Link className="text-sm text-cyan-300 hover:text-cyan-200" href="/events">Все события</Link>
          </div>
          {data?.recentEvents?.length ? (
            <div className="space-y-3">
              {data.recentEvents.map(e => (
                <Link className="block rounded-lg border border-slate-800 p-3 hover:bg-slate-800/60" href={`/events/${e.id}`} key={e.id}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{e.title}</div>
                      <div className="mt-1 text-sm text-slate-400">{e.camera?.name ?? 'Камера не указана'} · {formatDate(e.startedAt)}</div>
                    </div>
                    <Badge value={e.severity} tone={severityColors[e.severity]} />
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <EmptyState text="События еще не поступали" />
          )}
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Проблемные камеры</h2>
            <AlertTriangle className="h-5 w-5 text-amber-300" />
          </div>
          {data?.problemCameras?.length ? (
            <div className="space-y-3">
              {data.problemCameras.map(c => (
                <div className="rounded-lg border border-slate-800 p-3" key={c.id}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="font-medium">{c.name}</div>
                      <div className="mt-1 text-sm text-slate-400">{c.store?.name ?? 'Магазин не указан'} · {c.code}</div>
                    </div>
                    <Badge value={c.processingEnabled ? c.status : 'DISABLED'} tone={c.processingEnabled ? statusColors[c.status] : '#ef4444'} />
                  </div>
                  <div className="mt-2 text-xs text-slate-500">Последний кадр: {formatDate(c.lastFrameAt)}</div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState text="Нет камер, требующих внимания" />
          )}
        </section>
      </div>

      <section className="card mt-6">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-semibold">Analytics workers</h2>
          <CircleDot className="h-5 w-5 text-cyan-300" />
        </div>
        {data?.workers?.length ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            {data.workers.map(w => (
              <div className="rounded-lg border border-slate-800 p-3" key={w.id}>
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate font-medium">{w.name}</div>
                    <div className="truncate text-sm text-slate-400">{w.host}</div>
                  </div>
                  {w.status === 'ONLINE' || w.status === 'BUSY' ? <Wifi className="h-5 w-5 text-emerald-300" /> : <WifiOff className="h-5 w-5 text-slate-500" />}
                </div>
                <div className="mt-3 flex items-center justify-between gap-2 text-xs text-slate-400">
                  <Badge value={w.status} tone={statusColors[w.status]} />
                  <span>{formatDate(w.lastHeartbeatAt)}</span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState text="Worker-ы еще не зарегистрированы" />
        )}
      </section>
    </Layout>
  )
}
