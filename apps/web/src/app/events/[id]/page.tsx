'use client'

import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Camera, CheckCircle2, CreditCard, FileText, MessageSquareText, PackageSearch, Video } from 'lucide-react'
import { api } from '@/lib/api'
import Layout from '@/components/Layout'
import { useParams } from 'next/navigation'

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString('ru-RU') : '-'
}

function money(value?: string | number | null) {
  if (value === null || value === undefined) return '-'
  return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 }).format(Number(value))
}

function JsonBlock({ value }: { value: any }) {
  if (!value || Object.keys(value).length === 0) return <p className="muted">Нет данных</p>
  return <pre className="max-h-72 overflow-auto rounded-lg bg-slate-950 p-4 text-xs text-slate-300">{JSON.stringify(value, null, 2)}</pre>
}

function Section({ title, icon: Icon, children }: { title: string; icon: any; children: React.ReactNode }) {
  return (
    <section className="card">
      <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold">
        <Icon size={18} className="text-cyan-300" />
        {title}
      </h2>
      {children}
    </section>
  )
}

export default function EventDetailPage() {
  const { id } = useParams()
  const { data: e, isLoading } = useQuery({
    queryKey: ['event', id],
    queryFn: () => api.get(`/events/${id}`).then(r => r.data)
  })

  if (isLoading) return <Layout><p className="muted">Загрузка...</p></Layout>
  if (!e) return null

  const receipt = e.receipts
  const items = receipt?.receipt_items ?? []
  const evidence = e.event_evidence ?? []
  const transcripts = e.event_transcripts ?? []
  const scans = e.product_scans ?? []
  const observations = e.video_observations ?? []

  return (
    <Layout>
      <div className="mb-6 flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <div className="mb-2 flex flex-wrap gap-2 text-sm">
            <span className="rounded bg-slate-800 px-2 py-1 text-slate-300">{e.status}</span>
            <span className={e.severity === 'CRITICAL' ? 'rounded bg-red-950 px-2 py-1 text-red-200' : 'rounded bg-amber-950 px-2 py-1 text-amber-200'}>{e.severity}</span>
            {e.violation_types?.code && <span className="rounded bg-slate-800 px-2 py-1 text-slate-300">{e.violation_types.code}</span>}
          </div>
          <h1 className="text-3xl font-semibold">{e.title}</h1>
          <p className="muted mt-2">{e.description ?? e.event_types?.name ?? 'Описание не передано'}</p>
        </div>
        <div className="grid min-w-[280px] gap-2 text-sm text-slate-300">
          <div>{e.stores?.name ?? 'Магазин не указан'}</div>
          <div>{e.workplaces?.name ?? 'Рабочее место не указано'}</div>
          <div>{e.cameras?.name ?? 'Камера не указана'}</div>
          <div>{formatDate(e.started_at)}</div>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Section title="Итог" icon={AlertTriangle}>
          <dl className="grid gap-3 text-sm">
            <div><dt className="muted">Тип события</dt><dd>{e.event_types?.name ?? '-'}</dd></div>
            <div><dt className="muted">Нарушение</dt><dd>{e.violation_types?.name ?? '-'}</dd></div>
            <div><dt className="muted">Уверенность</dt><dd>{e.confidence ? `${Math.round(e.confidence * 100)}%` : '-'}</dd></div>
            <div><dt className="muted">Сумма риска</dt><dd>{money(e.risk_amount)}</dd></div>
          </dl>
        </Section>

        <Section title="Чек" icon={CreditCard}>
          {receipt ? (
            <dl className="grid gap-3 text-sm">
              <div><dt className="muted">Номер</dt><dd>{receipt.external_receipt_id}</dd></div>
              <div><dt className="muted">Оплата</dt><dd>{receipt.payment_method ?? '-'}</dd></div>
              <div><dt className="muted">Сумма</dt><dd>{money(receipt.receipt_total)}</dd></div>
              <div><dt className="muted">Время</dt><dd>{formatDate(receipt.occurred_at)}</dd></div>
            </dl>
          ) : <p className="muted">Чек не привязан</p>}
        </Section>

        <Section title="Видео" icon={Camera}>
          <dl className="grid gap-3 text-sm">
            <div><dt className="muted">Камера</dt><dd>{e.cameras?.name ?? '-'}</dd></div>
            <div><dt className="muted">Evidence</dt><dd>{evidence.length}</dd></div>
            <div><dt className="muted">Наблюдения</dt><dd>{observations.length}</dd></div>
          </dl>
        </Section>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-2">
        <Section title="Позиции чека" icon={FileText}>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[520px] text-left text-sm">
              <thead className="text-xs uppercase text-slate-500"><tr><th className="py-2">Товар</th><th>ШК</th><th>Кол-во</th><th>Сумма</th></tr></thead>
              <tbody>
                {items.map((item: any) => (
                  <tr key={item.id} className="border-t border-slate-800">
                    <td className="py-2">{item.product_name}</td>
                    <td>{item.barcode ?? '-'}</td>
                    <td>{money(item.quantity)}</td>
                    <td>{money(item.line_total)}</td>
                  </tr>
                ))}
                {!items.length && <tr><td className="py-6 text-center text-slate-500" colSpan={4}>Нет позиций</td></tr>}
              </tbody>
            </table>
          </div>
        </Section>

        <Section title="Сканы товаров" icon={PackageSearch}>
          <div className="grid gap-2">
            {scans.map((scan: any) => (
              <div key={scan.id} className="rounded-lg border border-slate-800 p-3 text-sm">
                <div className="font-medium">{scan.product_name ?? scan.barcode}</div>
                <div className="muted mt-1">{scan.barcode ?? '-'} · {formatDate(scan.occurred_at)}</div>
              </div>
            ))}
            {!scans.length && <p className="muted">Сканы не найдены</p>}
          </div>
        </Section>

        <Section title="Транскрипт" icon={MessageSquareText}>
          <div className="grid gap-3">
            {transcripts.map((line: any) => (
              <div key={line.id} className="rounded-lg bg-slate-950 p-3 text-sm">
                <div className="mb-1 text-xs text-slate-500">{line.speaker} · {formatDate(line.started_at)}</div>
                <div>{line.text}</div>
              </div>
            ))}
            {!transcripts.length && <p className="muted">Транскрипт не загружен</p>}
          </div>
        </Section>

        <Section title="Evidence" icon={Video}>
          <div className="grid gap-3">
            {evidence.map((x: any) => (
              <div key={x.id} className="rounded-lg border border-slate-800 p-3 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <span>{x.evidence_type}</span>
                  <span className="text-slate-500">{x.availability_status}</span>
                </div>
                <div className="muted mt-1">{formatDate(x.video_started_at)} - {formatDate(x.video_finished_at)}</div>
                {(x.public_url ?? x.file_path) && <a className="mt-2 block text-cyan-300" href={x.public_url ?? x.file_path} target="_blank">Открыть файл</a>}
                {x.evidence_type === 'IMAGE' && (x.public_url ?? x.file_path) && (
                  <img className="mt-3 max-h-72 rounded-lg object-contain" src={x.public_url ?? x.file_path} alt="Evidence" />
                )}
              </div>
            ))}
            {!evidence.length && <p className="muted">Evidence не привязан</p>}
          </div>
        </Section>

        <Section title="Наблюдения" icon={CheckCircle2}>
          <div className="grid gap-2">
            {observations.map((x: any) => (
              <div key={x.id} className="rounded-lg border border-slate-800 p-3 text-sm">
                <div>{x.observation_type} · {x.product_name ?? x.barcode ?? '-'}</div>
                <div className="muted mt-1">{formatDate(x.observed_at)} · {x.confidence ? `${Math.round(x.confidence * 100)}%` : '-'}</div>
              </div>
            ))}
            {!observations.length && <p className="muted">Наблюдения не найдены</p>}
          </div>
        </Section>

        <Section title="Metadata" icon={FileText}>
          <JsonBlock value={e.metadata} />
        </Section>
      </div>
    </Layout>
  )
}
