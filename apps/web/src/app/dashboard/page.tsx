'use client'

import Layout from '@/components/Layout'
import { api } from '@/lib/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Activity,
  AlertTriangle,
  Camera,
  CircleDot,
  Mic,
  Monitor,
  Plus,
  Radio,
  ReceiptText,
  RefreshCw,
  ScanLine,
  Store,
  UsersRound,
  Volume2,
  Wifi,
  WifiOff,
} from 'lucide-react'
import Link from 'next/link'
import type { FormEvent } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

type DashboardSummary = {
  totals: {
    stores: number
    workplaces: number
    cameras: number
    online: number
    offline: number
    streams: number
    workers: number
    receipts24h: number
    sessions24h: number
    observations24h: number
    transcripts24h: number
    events24h: number
    highRisk24h: number
    integrationErrorsOpen: number
  }
  cameraStatuses: { status: string; count: number }[]
  eventSeverities: { severity: string; count: number }[]
  eventsByDay: { date: string; count: number }[]
  topViolationTypes: { id: string; code?: string; name: string; riskLevel?: string; count: number }[]
  recentEvents: {
    id: string
    title: string
    severity: string
    status: string
    startedAt: string
    camera: { name: string; code: string } | null
    violationType: { name: string; code: string; riskLevel: string } | null
    receipt: { externalReceiptId: string; paymentMethod: string | null; total: string | null } | null
  }[]
  recentReceipts: {
    id: string
    externalReceiptId: string
    operationType: string
    receiptStatus: string
    paymentMethod: string | null
    total: string | null
    occurredAt: string
    store: { name: string; code: string } | null
    workplace: { name: string; externalId: string } | null
    employee: { name: string; externalId: string | null } | null
  }[]
  recentSessions: {
    id: string
    status: string
    startedAt: string
    finishedAt: string | null
    serviceScore: number | null
    customerPresent: boolean | null
    workplace: { name: string; externalId: string } | null
    employee: { name: string } | null
    receipt: { externalReceiptId: string; total: string | null; paymentMethod: string | null } | null
  }[]
  problemCameras: {
    id: string
    name: string
    code: string
    status: string
    processingEnabled: boolean
    lastFrameAt: string | null
    store: { name: string; code: string } | null
    workplace: { name: string; externalId: string } | null
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
  MEDIUM: '#f59e0b',
  INFO: '#06b6d4',
}

const statusColors: Record<string, string> = {
  ONLINE: '#22c55e',
  OFFLINE: '#64748b',
  ERROR: '#ef4444',
  BUSY: '#f59e0b',
  NEW: '#06b6d4',
  CONFIRMED: '#22c55e',
  FALSE_POSITIVE: '#64748b',
  RESOLVED: '#22c55e',
  unknown: '#64748b',
}

function field(form: HTMLFormElement, name: string) {
  return String(new FormData(form).get(name) ?? '').trim()
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

function money(value?: string | number | null) {
  if (value === undefined || value === null || value === '') return '-'
  return new Intl.NumberFormat('ru-KZ', { maximumFractionDigits: 0 }).format(Number(value))
}

function Badge({ value, tone }: { value: string; tone?: string }) {
  return (
    <span className="inline-flex h-7 items-center whitespace-nowrap rounded border border-slate-700 px-2.5 text-xs text-slate-200">
      <span className="mr-2 h-2 w-2 rounded-full" style={{ backgroundColor: tone ?? '#64748b' }} />
      {value}
    </span>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="rounded border border-dashed border-slate-800 p-6 text-center text-sm text-slate-400">{text}</div>
}

function Kpi({
  label,
  value,
  detail,
  icon: Icon,
  color,
}: {
  label: string
  value: number | string
  detail: string
  icon: typeof Activity
  color: string
}) {
  return (
    <div className="card min-h-28">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-sm text-slate-400">{label}</div>
          <div className="mt-2 text-2xl font-semibold">{value}</div>
          <div className="mt-2 truncate text-xs text-slate-500">{detail}</div>
        </div>
        <Icon className={`h-5 w-5 ${color}`} />
      </div>
    </div>
  )
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

  const createStore = useMutation({ mutationFn: (payload: Record<string, unknown>) => api.post('/resources/stores', payload), onSuccess: invalidateData })
  const createWorkplace = useMutation({ mutationFn: (payload: Record<string, unknown>) => api.post('/resources/workplaces', payload), onSuccess: invalidateData })
  const createCamera = useMutation({ mutationFn: (payload: Record<string, unknown>) => api.post('/resources/cameras', payload), onSuccess: invalidateData })
  const createStream = useMutation({ mutationFn: (payload: Record<string, unknown>) => api.post('/resources/streams', payload), onSuccess: invalidateData })

  const totals = data?.totals
  const cameraChart = data?.cameraStatuses?.length ? data.cameraStatuses : [{ status: 'нет данных', count: 1 }]
  const severityChart = data?.eventSeverities ?? []
  const trend = data?.eventsByDay ?? []

  return (
    <Layout>
      <div className="mb-6 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal">Главная панель</h1>
          <p className="mt-1 text-sm text-slate-400">Контроль продаж, сервиса, камер, чеков и обработки речи.</p>
        </div>
        <button className="inline-flex items-center gap-2 self-start md:self-auto" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
          {isFetching ? 'Обновление...' : 'Обновить'}
        </button>
      </div>

      {isError && (
        <div className="mb-6 rounded border border-red-900 bg-red-950/40 p-4 text-sm text-red-100">
          Не удалось загрузить дашборд. Проверь API и авторизацию.
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Kpi label="Чеки за 24ч" value={isLoading ? '-' : totals?.receipts24h ?? 0} detail={`${totals?.sessions24h ?? 0} sale sessions`} icon={ReceiptText} color="text-emerald-300" />
        <Kpi label="Нарушения за 24ч" value={isLoading ? '-' : totals?.events24h ?? 0} detail={`${totals?.highRisk24h ?? 0} высокого риска`} icon={AlertTriangle} color="text-amber-300" />
        <Kpi label="ИИ наблюдения" value={isLoading ? '-' : totals?.observations24h ?? 0} detail={`${totals?.transcripts24h ?? 0} транскриптов`} icon={ScanLine} color="text-cyan-300" />
        <Kpi label="Камеры" value={isLoading ? '-' : totals?.cameras ?? 0} detail={`${totals?.online ?? 0} онлайн, ${totals?.streams ?? 0} потоков`} icon={Camera} color="text-sky-300" />
      </div>

      <section className="card mt-6">
        <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-base font-semibold">Быстрое добавление</h2>
            <p className="mt-1 text-sm text-slate-400">Цепочка настройки: магазин, рабочее место, камера, видео и аудио поток.</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge value={`${totals?.stores ?? stores.length} магазинов`} tone="#22c55e" />
            <Badge value={`${totals?.workplaces ?? workplaces.length} рабочих мест`} tone="#38bdf8" />
            <Badge value={`${totals?.integrationErrorsOpen ?? 0} ошибок интеграции`} tone={totals?.integrationErrorsOpen ? '#ef4444' : '#64748b'} />
          </div>
        </div>
        <div className="grid gap-4 xl:grid-cols-5">
          <form className="rounded border border-slate-800 p-4" onSubmit={(event: FormEvent<HTMLFormElement>) => {
            event.preventDefault()
            const form = event.currentTarget
            createStore.mutate({ name: field(form, 'name'), code: field(form, 'code'), city: field(form, 'city') || undefined, address: field(form, 'address') || undefined }, { onSuccess: () => form.reset() })
          }}>
            <div className="mb-3 flex items-center gap-2 font-medium"><Store className="h-4 w-4 text-emerald-300" />Магазин</div>
            <div className="space-y-3">
              <input className="w-full" name="name" placeholder="Название" required />
              <input className="w-full" name="code" placeholder="Код, например store-1" required />
              <input className="w-full" name="city" placeholder="Город" />
              <input className="w-full" name="address" placeholder="Адрес" />
              <button className="w-full" disabled={createStore.isPending}>{createStore.isPending ? 'Сохранение...' : 'Добавить'}</button>
            </div>
          </form>

          <form className="rounded border border-slate-800 p-4" onSubmit={(event: FormEvent<HTMLFormElement>) => {
            event.preventDefault()
            const form = event.currentTarget
            createWorkplace.mutate({ store_id: field(form, 'store_id'), name: field(form, 'name'), external_id: field(form, 'external_id'), workplace_type: 'checkout', is_active: true }, { onSuccess: () => form.reset() })
          }}>
            <div className="mb-3 flex items-center gap-2 font-medium"><Monitor className="h-4 w-4 text-sky-300" />Рабочее место</div>
            <div className="space-y-3">
              <select className="w-full" name="store_id" required defaultValue="">
                <option value="" disabled>Магазин</option>
                {stores.map(s => <option key={s.id} value={s.id}>{s.name} · {s.code}</option>)}
              </select>
              <input className="w-full" name="name" placeholder="Касса 1" required />
              <input className="w-full" name="external_id" placeholder="ID из 1С, pos-1" required />
              <button className="w-full" disabled={createWorkplace.isPending || !stores.length}>{createWorkplace.isPending ? 'Сохранение...' : 'Добавить'}</button>
            </div>
          </form>

          <form className="rounded border border-slate-800 p-4" onSubmit={(event: FormEvent<HTMLFormElement>) => {
            event.preventDefault()
            const form = event.currentTarget
            createCamera.mutate({ workplace_id: field(form, 'workplace_id'), name: field(form, 'name'), code: field(form, 'code'), location_description: field(form, 'location_description') || undefined, processing_enabled: true, is_active: true }, { onSuccess: () => form.reset() })
          }}>
            <div className="mb-3 flex items-center gap-2 font-medium"><Camera className="h-4 w-4 text-cyan-300" />Камера</div>
            <div className="space-y-3">
              <select className="w-full" name="workplace_id" required defaultValue="">
                <option value="" disabled>Рабочее место</option>
                {workplaces.map(w => <option key={w.id} value={w.id}>{w.name} · {w.external_id}</option>)}
              </select>
              <input className="w-full" name="name" placeholder="Название камеры" required />
              <input className="w-full" name="code" placeholder="checkout-1" required />
              <input className="w-full" name="location_description" placeholder="Расположение" />
              <button className="w-full" disabled={createCamera.isPending || !workplaces.length}>{createCamera.isPending ? 'Сохранение...' : 'Добавить'}</button>
            </div>
          </form>

          {(['RTSP_VIDEO', 'RTSP_AUDIO'] as const).map(type => (
            <form className="rounded border border-slate-800 p-4" key={type} onSubmit={(event: FormEvent<HTMLFormElement>) => {
              event.preventDefault()
              const form = event.currentTarget
              createStream.mutate({ camera_id: field(form, 'camera_id'), stream_type: type, stream_url: field(form, 'stream_url'), transport: 'tcp', is_primary: type === 'RTSP_VIDEO', is_enabled: true }, { onSuccess: () => form.reset() })
            }}>
              <div className="mb-3 flex items-center gap-2 font-medium">
                {type === 'RTSP_VIDEO' ? <Radio className="h-4 w-4 text-violet-300" /> : <Mic className="h-4 w-4 text-amber-300" />}
                {type === 'RTSP_VIDEO' ? 'Видео RTSP' : 'Аудио RTSP'}
              </div>
              <div className="space-y-3">
                <select className="w-full" name="camera_id" required defaultValue="">
                  <option value="" disabled>Камера</option>
                  {cameras.map(c => <option key={c.id} value={c.id}>{c.name} · {c.code}</option>)}
                </select>
                <input className="w-full" name="stream_url" placeholder={type === 'RTSP_VIDEO' ? 'rtsp://.../video' : 'rtsp://.../audio'} required />
                <button className="w-full" disabled={createStream.isPending || !cameras.length}>{createStream.isPending ? 'Сохранение...' : 'Добавить'}</button>
              </div>
            </form>
          ))}
        </div>
        {(createStore.isError || createWorkplace.isError || createCamera.isError || createStream.isError) && (
          <div className="mt-4 rounded border border-red-900 bg-red-950/40 p-3 text-sm text-red-100">Не удалось сохранить. Проверь обязательные поля и права пользователя.</div>
        )}
      </section>

      <div className="mt-6 grid gap-4 xl:grid-cols-[1fr_1fr_1fr]">
        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Статусы камер</h2>
            <Badge value={`${totals?.offline ?? 0} офлайн`} tone={statusColors.OFFLINE} />
          </div>
          <div className="h-64">
            <ResponsiveContainer>
              <PieChart>
                <Pie data={cameraChart} dataKey="count" nameKey="status" innerRadius={50} outerRadius={86} paddingAngle={2}>
                  {cameraChart.map(x => <Cell key={x.status} fill={statusColors[x.status] ?? statusColors.unknown} />)}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Severity за 7 дней</h2>
            <Badge value="нарушения" tone="#f59e0b" />
          </div>
          {severityChart.length ? (
            <div className="h-64">
              <ResponsiveContainer>
                <BarChart data={severityChart}>
                  <CartesianGrid stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="severity" stroke="#94a3b8" />
                  <YAxis allowDecimals={false} stroke="#94a3b8" />
                  <Tooltip cursor={{ fill: '#0f172a' }} />
                  <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                    {severityChart.map(x => <Cell key={x.severity} fill={severityColors[x.severity] ?? '#64748b'} />)}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          ) : <EmptyState text="За последние 7 дней событий нет" />}
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Динамика</h2>
            <Badge value="7 дней" tone="#06b6d4" />
          </div>
          {trend.length ? (
            <div className="h-64">
              <ResponsiveContainer>
                <LineChart data={trend}>
                  <CartesianGrid stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="date" stroke="#94a3b8" tickFormatter={x => String(x).slice(5)} />
                  <YAxis allowDecimals={false} stroke="#94a3b8" />
                  <Tooltip />
                  <Line type="monotone" dataKey="count" stroke="#22c55e" strokeWidth={2} dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          ) : <EmptyState text="Нет данных для графика" />}
        </section>
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-2">
        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Последние нарушения</h2>
            <Link className="text-sm text-cyan-300 hover:text-cyan-200" href="/events">Все события</Link>
          </div>
          {data?.recentEvents?.length ? (
            <div className="space-y-3">
              {data.recentEvents.map(e => (
                <Link className="block rounded border border-slate-800 p-3 hover:bg-slate-800/60" href={`/events/${e.id}`} key={e.id}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{e.violationType?.name ?? e.title}</div>
                      <div className="mt-1 text-sm text-slate-400">{e.camera?.name ?? 'Камера не указана'} · {e.receipt?.externalReceiptId ?? 'чек не привязан'} · {formatDate(e.startedAt)}</div>
                    </div>
                    <Badge value={e.severity} tone={severityColors[e.severity]} />
                  </div>
                </Link>
              ))}
            </div>
          ) : <EmptyState text="Нарушения еще не поступали" />}
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Последние чеки</h2>
            <Badge value={`${totals?.receipts24h ?? 0} за 24ч`} tone="#22c55e" />
          </div>
          {data?.recentReceipts?.length ? (
            <div className="space-y-3">
              {data.recentReceipts.map(r => (
                <div className="rounded border border-slate-800 p-3" key={r.id}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{r.externalReceiptId}</div>
                      <div className="mt-1 text-sm text-slate-400">{r.workplace?.name ?? 'Касса не указана'} · {r.employee?.name ?? 'сотрудник не указан'} · {formatDate(r.occurredAt)}</div>
                    </div>
                    <div className="text-right">
                      <div className="font-semibold">{money(r.total)}</div>
                      <div className="mt-1 text-xs text-slate-500">{r.paymentMethod ?? '-'}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : <EmptyState text="Чеки еще не поступали" />}
        </section>
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-3">
        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Sale sessions</h2>
            <Badge value={`${totals?.sessions24h ?? 0} за 24ч`} tone="#38bdf8" />
          </div>
          {data?.recentSessions?.length ? (
            <div className="space-y-3">
              {data.recentSessions.map(s => (
                <div className="rounded border border-slate-800 p-3" key={s.id}>
                  <div className="flex justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{s.receipt?.externalReceiptId ?? `Сессия ${s.id}`}</div>
                      <div className="mt-1 text-sm text-slate-400">{s.workplace?.name ?? 'Касса'} · {formatDate(s.startedAt)}</div>
                    </div>
                    <Badge value={s.status} tone={statusColors[s.status]} />
                  </div>
                  <div className="mt-2 text-xs text-slate-500">Клиент: {s.customerPresent === null ? 'не определено' : s.customerPresent ? 'да' : 'нет'} · сервис: {s.serviceScore ?? '-'}</div>
                </div>
              ))}
            </div>
          ) : <EmptyState text="Sale sessions еще нет" />}
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Проблемные камеры</h2>
            <AlertTriangle className="h-5 w-5 text-amber-300" />
          </div>
          {data?.problemCameras?.length ? (
            <div className="space-y-3">
              {data.problemCameras.map(c => (
                <div className="rounded border border-slate-800 p-3" key={c.id}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{c.name}</div>
                      <div className="mt-1 text-sm text-slate-400">{c.workplace?.name ?? c.store?.name ?? 'Место не указано'} · {c.code}</div>
                    </div>
                    <Badge value={c.processingEnabled ? c.status : 'DISABLED'} tone={c.processingEnabled ? statusColors[c.status] : '#ef4444'} />
                  </div>
                  <div className="mt-2 text-xs text-slate-500">Последний кадр: {formatDate(c.lastFrameAt)}</div>
                </div>
              ))}
            </div>
          ) : <EmptyState text="Нет камер, требующих внимания" />}
        </section>

        <section className="card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Worker-ы</h2>
            <CircleDot className="h-5 w-5 text-cyan-300" />
          </div>
          {data?.workers?.length ? (
            <div className="space-y-3">
              {data.workers.map(w => (
                <div className="rounded border border-slate-800 p-3" key={w.id}>
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{w.name}</div>
                      <div className="truncate text-sm text-slate-400">{w.host}</div>
                    </div>
                    {w.status === 'ONLINE' || w.status === 'BUSY' ? <Wifi className="h-5 w-5 text-emerald-300" /> : <WifiOff className="h-5 w-5 text-slate-500" />}
                  </div>
                  <div className="mt-2 flex items-center justify-between gap-2 text-xs text-slate-400">
                    <Badge value={w.status} tone={statusColors[w.status]} />
                    <span>{formatDate(w.lastHeartbeatAt)}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : <EmptyState text="Worker-ы еще не зарегистрированы" />}
        </section>
      </div>

      {data?.topViolationTypes?.length ? (
        <section className="card mt-6">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold">Топ нарушений за 7 дней</h2>
            <Volume2 className="h-5 w-5 text-slate-400" />
          </div>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {data.topViolationTypes.map(v => (
              <div className="rounded border border-slate-800 p-3" key={v.id}>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate font-medium">{v.name}</div>
                    <div className="mt-1 text-xs text-slate-500">{v.code}</div>
                  </div>
                  <div className="text-lg font-semibold">{v.count}</div>
                </div>
              </div>
            ))}
          </div>
        </section>
      ) : null}
    </Layout>
  )
}
