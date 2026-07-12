import { Controller, Get, Query } from '@nestjs/common';
import type { AuthUser } from '@cashier/shared';
import { CurrentUser } from '../common/security';
import { PrismaService } from '../common/prisma.service';

@Controller()
export class DashboardController {
  constructor(private db: PrismaService) {}

  private orgId(user: AuthUser) {
    return user.organizationId ? BigInt(user.organizationId) : null;
  }

  private orgWhere(user: AuthUser) {
    const orgId = this.orgId(user);
    return user.role === 'SUPER_ADMIN' ? {} : orgId ? { organization_id: orgId } : { organization_id: BigInt(-1) };
  }

  private cameraWhere(user: AuthUser) {
    const orgId = this.orgId(user);
    return user.role === 'SUPER_ADMIN' ? {} : orgId ? { stores: { organization_id: orgId } } : { id: BigInt(-1) };
  }

  private json(value: any): any {
    if (typeof value === 'bigint') return String(value);
    if (value instanceof Date) return value;
    if (Array.isArray(value)) return value.map(v => this.json(v));
    if (value && typeof value === 'object') {
      return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, this.json(v)]));
    }
    return value;
  }

  @Get('dashboard/summary')
  async summary(@CurrentUser() user: AuthUser) {
    const orgScope = this.orgWhere(user);
    const cameraScope = this.cameraWhere(user);
    const since24h = new Date(Date.now() - 864e5);
    const since7d = new Date(Date.now() - 7 * 864e5);

    const [
      stores,
      workplaces,
      cameras,
      online,
      streams,
      workers,
      receipts24h,
      sessions24h,
      observations24h,
      transcripts24h,
      events24h,
      highRisk24h,
      integrationErrorsOpen,
      cameraStatuses,
      eventSeverities,
      eventsByDay,
      recentEvents,
      recentReceipts,
      recentSessions,
      problemCameras,
      recentWorkers,
      topViolationTypes,
    ] = await this.db.$transaction([
      this.db.stores.count({ where: orgScope }),
      this.db.workplaces.count({ where: user.role === 'SUPER_ADMIN' ? {} : { stores: orgScope } }),
      this.db.cameras.count({ where: cameraScope }),
      this.db.cameras.count({ where: { ...cameraScope, status: 'ONLINE' } }),
      this.db.camera_streams.count({ where: user.role === 'SUPER_ADMIN' ? {} : { cameras: cameraScope } }),
      this.db.analytics_workers.count({ where: orgScope }),
      this.db.receipts.count({ where: { ...orgScope, occurred_at: { gte: since24h } } }),
      this.db.sale_sessions.count({ where: { ...orgScope, started_at: { gte: since24h } } }),
      this.db.video_observations.count({ where: { ...orgScope, observed_at: { gte: since24h } } }),
      this.db.event_transcripts.count({ where: { ...orgScope, started_at: { gte: since24h } } }),
      this.db.analytics_events.count({ where: { ...orgScope, created_at: { gte: since24h } } }),
      this.db.analytics_events.count({ where: { ...orgScope, severity: 'CRITICAL', created_at: { gte: since24h } } }),
      this.db.integration_errors.count({ where: { ...orgScope, status: 'OPEN' } }),
      this.db.cameras.groupBy({ by: ['status'], where: cameraScope, orderBy: { status: 'asc' }, _count: true }),
      this.db.analytics_events.groupBy({ by: ['severity'], where: { ...orgScope, started_at: { gte: since7d } }, orderBy: { severity: 'asc' }, _count: true }),
      this.db.analytics_events.groupBy({ by: ['started_at'], where: { ...orgScope, started_at: { gte: since7d } }, orderBy: { started_at: 'asc' }, _count: true }),
      this.db.analytics_events.findMany({
        where: orgScope,
        take: 8,
        orderBy: { started_at: 'desc' },
        include: {
          cameras: { select: { name: true, code: true } },
          event_types: { select: { name: true, code: true } },
          violation_types: { select: { name: true, code: true, risk_level: true } },
          receipts: { select: { external_receipt_id: true, payment_method: true, receipt_total: true } },
        },
      }),
      this.db.receipts.findMany({
        where: orgScope,
        take: 8,
        orderBy: { occurred_at: 'desc' },
        include: {
          stores: { select: { name: true, code: true } },
          workplaces: { select: { name: true, external_id: true } },
          employees: { select: { full_name: true, external_id: true } },
        },
      }),
      this.db.sale_sessions.findMany({
        where: orgScope,
        take: 8,
        orderBy: { started_at: 'desc' },
        include: {
          receipts: { select: { external_receipt_id: true, receipt_total: true, payment_method: true } },
          workplaces: { select: { name: true, external_id: true } },
          employees: { select: { full_name: true } },
        },
      }),
      this.db.cameras.findMany({
        where: { ...cameraScope, OR: [{ status: { not: 'ONLINE' } }, { processing_enabled: false }] },
        take: 8,
        orderBy: { updated_at: 'desc' },
        include: {
          stores: { select: { name: true, code: true } },
          workplaces: { select: { name: true, external_id: true } },
        },
      }),
      this.db.analytics_workers.findMany({ where: orgScope, take: 8, orderBy: { updated_at: 'desc' } }),
      this.db.analytics_events.groupBy({
        by: ['violation_type_id'],
        where: { ...orgScope, violation_type_id: { not: null }, started_at: { gte: since7d } },
        orderBy: { _count: { violation_type_id: 'desc' } },
        take: 6,
        _count: true,
      }),
    ]);

    const violationIds = topViolationTypes.map(x => x.violation_type_id).filter((x): x is bigint => x !== null);
    const violations = violationIds.length
      ? await this.db.violation_types.findMany({ where: { id: { in: violationIds } } })
      : [];
    const violationById = new Map(violations.map(x => [String(x.id), x]));

    return this.json({
      totals: {
        stores,
        workplaces,
        cameras,
        online,
        offline: cameras - online,
        streams,
        workers,
        receipts24h,
        sessions24h,
        observations24h,
        transcripts24h,
        events24h,
        highRisk24h,
        integrationErrorsOpen,
      },
      cameraStatuses: cameraStatuses.map(x => ({ status: x.status, count: x._count })),
      eventSeverities: eventSeverities.map(x => ({ severity: x.severity, count: x._count })),
      eventsByDay: Object.values(eventsByDay.reduce((acc: any, row: any) => {
        const key = row.started_at.toISOString().slice(0, 10);
        acc[key] ??= { date: key, count: 0 };
        acc[key].count += row._count;
        return acc;
      }, {})),
      topViolationTypes: topViolationTypes.map(x => {
        const v = violationById.get(String(x.violation_type_id));
        return { id: String(x.violation_type_id), code: v?.code, name: v?.name ?? 'Без типа', riskLevel: v?.risk_level, count: x._count };
      }),
      recentEvents: recentEvents.map(e => ({
        id: String(e.id),
        title: e.title,
        severity: e.severity,
        status: e.status,
        startedAt: e.started_at,
        camera: e.cameras ? { name: e.cameras.name, code: e.cameras.code } : null,
        eventType: e.event_types ? { name: e.event_types.name, code: e.event_types.code } : null,
        violationType: e.violation_types ? { name: e.violation_types.name, code: e.violation_types.code, riskLevel: e.violation_types.risk_level } : null,
        receipt: e.receipts ? { externalReceiptId: e.receipts.external_receipt_id, paymentMethod: e.receipts.payment_method, total: e.receipts.receipt_total } : null,
      })),
      recentReceipts: recentReceipts.map(r => ({
        id: String(r.id),
        externalReceiptId: r.external_receipt_id,
        operationType: r.operation_type,
        receiptStatus: r.receipt_status,
        paymentMethod: r.payment_method,
        total: r.receipt_total,
        occurredAt: r.occurred_at,
        store: r.stores ? { name: r.stores.name, code: r.stores.code } : null,
        workplace: r.workplaces ? { name: r.workplaces.name, externalId: r.workplaces.external_id } : null,
        employee: r.employees ? { name: r.employees.full_name, externalId: r.employees.external_id } : null,
      })),
      recentSessions: recentSessions.map(s => ({
        id: String(s.id),
        status: s.status,
        startedAt: s.started_at,
        finishedAt: s.finished_at,
        serviceScore: s.service_score,
        customerPresent: s.customer_present,
        workplace: s.workplaces ? { name: s.workplaces.name, externalId: s.workplaces.external_id } : null,
        employee: s.employees ? { name: s.employees.full_name } : null,
        receipt: s.receipts ? { externalReceiptId: s.receipts.external_receipt_id, total: s.receipts.receipt_total, paymentMethod: s.receipts.payment_method } : null,
      })),
      problemCameras: problemCameras.map(c => ({
        id: String(c.id),
        name: c.name,
        code: c.code,
        status: c.status,
        processingEnabled: c.processing_enabled,
        lastFrameAt: c.last_frame_at,
        store: c.stores ? { name: c.stores.name, code: c.stores.code } : null,
        workplace: c.workplaces ? { name: c.workplaces.name, externalId: c.workplaces.external_id } : null,
      })),
      workers: recentWorkers.map(w => ({
        id: String(w.id),
        name: w.name,
        host: w.host,
        version: w.version,
        status: w.status,
        lastHeartbeatAt: w.last_heartbeat_at,
      })),
    });
  }

  @Get('reports/events')
  async report(@CurrentUser() user: AuthUser, @Query() query: any) {
    return this.db.analytics_events.groupBy({
      by: ['severity'],
      where: {
        ...this.orgWhere(user),
        started_at: {
          gte: new Date(query.from ?? Date.now() - 7 * 864e5),
          lte: new Date(query.to ?? Date.now()),
        },
      },
      orderBy: { severity: 'asc' },
      _count: true,
    });
  }
}
