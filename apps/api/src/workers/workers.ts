import { Body, Controller, Get, Post, Req, UseGuards } from '@nestjs/common';
import { PrismaService } from '../common/prisma.service';
import { Public, WorkerGuard, maskStreamUrl } from '../common/security';

@Public()
@UseGuards(WorkerGuard)
@Controller('workers')
export class WorkersController {
  constructor(private db: PrismaService) {}

  @Post('heartbeat')
  heartbeat(@Req() r: any, @Body() d: any) {
    return this.db.analytics_workers.update({
      where: { id: r.worker.id },
      data: { last_heartbeat_at: new Date(), status: d.status ?? 'ONLINE', metadata: d.metadata ?? r.worker.metadata },
    });
  }

  @Get('me/config')
  config(@Req() r: any) {
    return { worker: r.worker, heartbeatIntervalSeconds: 30 };
  }

  @Get('me/cameras')
  async cameras(@Req() r: any) {
    const assignments = await this.db.worker_camera_assignments.findMany({
      where: { worker_id: r.worker.id, is_enabled: true },
      include: {
        cameras: {
          include: {
            camera_streams: true,
            camera_rois: { include: { roi_types: true } },
            camera_models: { include: { model_versions: true } },
          },
        },
      },
    });
    return assignments.map((x: any) => ({
      ...x,
      camera: {
        ...x.cameras,
        camera_streams: x.cameras.camera_streams.map((s: any) => ({ ...s, stream_url: maskStreamUrl(s.stream_url) })),
      },
    }));
  }

  @Post('me/sessions')
  session(@Req() r: any, @Body() d: any) {
    return this.db.processing_sessions.create({
      data: { ...d, worker_id: r.worker.id, started_at: new Date(d.startedAt), metadata: d.metadata ?? {} },
    });
  }

  @Post('me/metrics')
  async metric(@Req() r: any, @Body() d: any) {
    const recent = await this.db.camera_metrics.findFirst({
      where: { camera_id: BigInt(d.cameraId), recorded_at: { gt: new Date(Date.now() - 30000) } },
    });
    if (recent) return { accepted: false, reason: 'rate_limited' };
    return this.db.camera_metrics.create({
      data: {
        ...d,
        camera_id: BigInt(d.cameraId),
        worker_id: r.worker.id,
        recorded_at: new Date(d.recordedAt ?? Date.now()),
        metadata: d.metadata ?? {},
      },
    });
  }

  @Post('me/errors')
  async error(@Req() r: any, @Body() d: any) {
    const camera = d.cameraId
      ? await this.db.cameras.findFirst({
        where: { id: BigInt(d.cameraId), stores: { organization_id: r.worker.organization_id } },
        select: { id: true, store_id: true, workplace_id: true },
      })
      : null;
    return this.db.integration_errors.create({
      data: {
        organization_id: r.worker.organization_id,
        store_id: camera?.store_id,
        workplace_id: camera?.workplace_id,
        source_system: d.sourceSystem ?? 'PYTHON_WORKER',
        entity_type: d.entityType ?? 'WORKER',
        external_id: d.externalId ?? String(r.worker.id),
        error_code: d.errorCode ?? d.code ?? 'WORKER_ERROR',
        error_message: d.errorMessage ?? d.message ?? 'Worker error',
        payload: { ...(d.payload ?? {}), workerId: String(r.worker.id), cameraId: d.cameraId },
        status: 'OPEN',
        occurred_at: d.occurredAt ? new Date(d.occurredAt) : new Date(),
      },
    });
  }

  @Post('me/logs')
  logs(@Req() r: any, @Body() d: any) {
    return { accepted: true, workerId: r.worker.id, count: Array.isArray(d) ? d.length : 1 };
  }
}
