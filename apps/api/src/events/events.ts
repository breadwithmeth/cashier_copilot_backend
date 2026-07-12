import { BadRequestException, Body, Controller, Get, Param, Patch, Post, Query, Req, UseGuards } from '@nestjs/common';
import { IsArray, IsDateString, IsOptional, IsString } from 'class-validator';
import type { AuthUser } from '@cashier/shared';
import { PrismaService } from '../common/prisma.service';
import { CurrentUser, Public, Roles, WorkerGuard, tenantWhere } from '../common/security';
import { RealtimeGateway } from '../realtime/realtime';

class EventDto {
  @IsString()
  cameraId!: string;

  @IsString()
  eventTypeCode!: string;

  @IsDateString()
  startedAt!: string;

  @IsOptional()
  @IsString()
  deduplicationKey?: string;

  @IsString()
  title!: string;

  @IsOptional()
  description?: string;

  @IsOptional()
  severity?: 'INFO' | 'WARNING' | 'CRITICAL';

  @IsOptional()
  confidence?: number;

  @IsOptional()
  metadata?: any;

  @IsOptional()
  @IsArray()
  objects?: any[];

  @IsOptional()
  @IsArray()
  transcripts?: any[];

  @IsOptional()
  @IsArray()
  evidence?: any[];
}

export class EventsService {
  constructor(private db: PrismaService, private ws: RealtimeGateway) {}

  private async resolveViolation(tx: any, organizationId: bigint, code?: string | null) {
    if (!code) return null;
    return tx.violation_types.findFirst({
      where: {
        code,
        is_active: true,
        OR: [{ organization_id: organizationId }, { organization_id: null }],
      },
      orderBy: { organization_id: 'desc' },
      select: { id: true, risk_level: true, name: true },
    });
  }

  private inferredViolationCode(d: EventDto) {
    if (d.eventTypeCode === 'PRODUCT_SCANNED' && d.metadata?.customerPresent === false) {
      return 'PRODUCT_SCANNED_WITHOUT_CUSTOMER';
    }
    if (d.metadata?.customerPresent === true && d.metadata?.receiptPresent === false) {
      return 'CUSTOMER_WITHOUT_RECEIPT';
    }
    if (d.metadata?.productGiven === true && d.metadata?.paid === false) {
      return 'PRODUCT_GIVEN_WITHOUT_PAYMENT';
    }
    if (d.metadata?.receiptPresent === true && d.metadata?.customerPresent === false) {
      return 'RECEIPT_WITHOUT_CUSTOMER';
    }
    if (d.metadata?.receivingMismatch === true) {
      return 'RECEIVING_MISMATCH';
    }
    return d.metadata?.violationCode;
  }

  private severityForRisk(risk?: string | null) {
    if (risk === 'HIGH') return 'CRITICAL';
    if (risk === 'MEDIUM') return 'WARNING';
    return 'INFO';
  }

  async ingest(worker: any, d: EventDto) {
    const camera = await this.db.cameras.findFirst({
      where: { id: BigInt(d.cameraId), stores: { organization_id: worker.organization_id } },
      include: { stores: true },
    });
    if (!camera) throw new BadRequestException('Камера недоступна');

    if (d.deduplicationKey) {
      const old = await this.db.analytics_events.findFirst({ where: { deduplication_key: d.deduplicationKey } });
      if (old) return old;
    }

    const type = await this.db.event_types.findUnique({ where: { code: d.eventTypeCode } });
    if (!type) throw new BadRequestException('Неизвестный тип события');

    const event = await this.db.$transaction(async (tx: any) => {
      const violationCode = this.inferredViolationCode(d);
      const violation = await this.resolveViolation(tx, worker.organization_id, violationCode);
      const severity = d.severity ?? (violation ? this.severityForRisk(violation.risk_level) : type.default_severity);
      const metadata = { ...(d.metadata ?? {}), ...(violationCode ? { violationCode } : {}) };
      const receipt = metadata.externalReceiptId
        ? await tx.receipts.findUnique({
          where: { organization_id_external_receipt_id: { organization_id: worker.organization_id, external_receipt_id: metadata.externalReceiptId } },
          select: { id: true },
        })
        : null;
      const saleSession = receipt ? await tx.sale_sessions.findUnique({ where: { receipt_id: receipt.id }, select: { id: true } }) : null;

      const e = await tx.analytics_events.create({
        data: {
          organization_id: worker.organization_id,
          store_id: camera.store_id,
          workplace_id: camera.workplace_id,
          camera_id: camera.id,
          event_type_id: type.id,
          worker_id: worker.id,
          started_at: new Date(d.startedAt),
          severity,
          title: violation?.name ?? d.title,
          description: d.description,
          confidence: d.confidence,
          metadata,
          deduplication_key: d.deduplicationKey,
          external_receipt_id: metadata.externalReceiptId,
          receipt_id: receipt?.id,
          sale_session_id: saleSession?.id,
          violation_type_id: violation?.id,
          risk_amount: metadata.riskAmount,
          operation_type: metadata.operationType,
          event_objects: { create: d.objects ?? [] },
          event_transcripts: {
            create: (d.transcripts ?? []).map(x => ({
              ...x,
              organization_id: worker.organization_id,
              store_id: camera.store_id,
              workplace_id: camera.workplace_id,
              camera_id: camera.id,
              receipt_id: receipt?.id,
              sale_session_id: saleSession?.id,
              started_at: new Date(x.startedAt),
              finished_at: x.finishedAt ? new Date(x.finishedAt) : null,
              metadata: x.metadata ?? {},
            })),
          },
          event_evidence: {
            create: (d.evidence ?? []).map(x => ({
              ...x,
              camera_id: camera.id,
              receipt_id: receipt?.id,
              captured_at: new Date(x.capturedAt),
              expires_at: x.expiresAt ? new Date(x.expiresAt) : null,
              video_started_at: x.videoStartedAt ? new Date(x.videoStartedAt) : null,
              video_finished_at: x.videoFinishedAt ? new Date(x.videoFinishedAt) : null,
              pre_seconds: x.preSeconds,
              post_seconds: x.postSeconds,
              metadata: x.metadata ?? {},
            })),
          },
        },
      });
      await tx.notifications.create({ data: { event_id: e.id, channel: 'WEBSOCKET', recipient: `org:${worker.organization_id}`, payload: { eventId: e.id } } });
      return e;
    });

    this.ws.emitOrg(String(worker.organization_id), 'analytics-event.created', event);
    return event;
  }
}

@Controller('events')
export class EventsController {
  constructor(private db: PrismaService) {}

  private json(value: any): any {
    if (typeof value === 'bigint') return String(value);
    if (value instanceof Date) return value;
    if (Array.isArray(value)) return value.map(v => this.json(v));
    if (value && typeof value === 'object') return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, this.json(v)]));
    return value;
  }

  @Get()
  async list(@CurrentUser() u: AuthUser, @Query() q: any) {
    const page = Math.max(1, +q.page || 1);
    const take = Math.min(100, +q.pageSize || 25);
    const where: any = {
      ...tenantWhere(u),
      ...(q.status && { status: q.status }),
      ...(q.cameraId && { camera_id: BigInt(q.cameraId) }),
      ...(q.from || q.to ? { started_at: { ...(q.from && { gte: new Date(q.from) }), ...(q.to && { lte: new Date(q.to) }) } } : {}),
    };
    const [data, total] = await this.db.$transaction([
      this.db.analytics_events.findMany({
        where,
        skip: (page - 1) * take,
        take,
        orderBy: { started_at: q.order === 'asc' ? 'asc' : 'desc' },
        include: { cameras: true, event_types: true, violation_types: true, receipts: true },
      }),
      this.db.analytics_events.count({ where }),
    ]);
    return this.json({ data, total, page, pageSize: take });
  }

  @Get(':id')
  async one(@CurrentUser() u: AuthUser, @Param('id') id: string) {
    const event = await this.db.analytics_events.findFirstOrThrow({
      where: { id: BigInt(id), ...tenantWhere(u) },
      include: {
        stores: true,
        workplaces: true,
        cameras: { include: { camera_streams: true } },
        event_types: true,
        violation_types: true,
        receipts: { include: { receipt_items: true } },
        sale_sessions: { include: { service_check_results: true } },
        event_objects: true,
        event_transcripts: true,
        event_evidence: true,
        event_reviews: { include: { employees: { select: { full_name: true } } } },
      },
    });
    const productScans = event.receipt_id
      ? await this.db.product_scans.findMany({ where: { receipt_id: event.receipt_id }, orderBy: { occurred_at: 'asc' } })
      : event.external_receipt_id
        ? await this.db.product_scans.findMany({ where: { organization_id: event.organization_id, external_receipt_id: event.external_receipt_id }, orderBy: { occurred_at: 'asc' } })
        : [];
    const observations = await this.db.video_observations.findMany({
      where: {
        organization_id: event.organization_id,
        OR: [
          { sale_session_id: event.sale_session_id ?? BigInt(-1) },
          { receipt_id: event.receipt_id ?? BigInt(-1) },
          { camera_id: event.camera_id ?? BigInt(-1), observed_at: { gte: event.started_at, lte: event.finished_at ?? new Date(event.started_at.getTime() + 120000) } },
        ],
      },
      orderBy: { observed_at: 'asc' },
      take: 100,
    });
    return this.json({ ...event, product_scans: productScans, video_observations: observations });
  }

  @Roles('OPERATOR', 'MANAGER', 'ORGANIZATION_ADMIN', 'SUPER_ADMIN')
  @Patch(':id/status')
  async status(@CurrentUser() u: AuthUser, @Param('id') id: string, @Body() d: { status: any; comment?: string }) {
    const e = await this.db.analytics_events.findFirstOrThrow({ where: { id: BigInt(id), ...tenantWhere(u) } });
    const row = await this.db.$transaction(async (tx: any) => {
      const x = await tx.analytics_events.update({ where: { id: BigInt(id) }, data: { status: d.status } });
      await tx.event_reviews.create({
        data: {
          event_id: BigInt(id),
          reviewer_id: BigInt(u.sub),
          decision: d.status,
          comment: d.comment,
          previous_status: e.status,
          new_status: d.status,
        },
      });
      return x;
    });
    return this.json(row);
  }
}

@Public()
@UseGuards(WorkerGuard)
@Controller('workers/me/events')
export class WorkerEventsController {
  constructor(private svc: EventsService) {}

  @Post()
  one(@Req() r: any, @Body() d: EventDto) {
    return this.svc.ingest(r.worker, d);
  }

  @Post('batch')
  batch(@Req() r: any, @Body() d: EventDto[]) {
    return Promise.all(d.map(x => this.svc.ingest(r.worker, x)));
  }
}
