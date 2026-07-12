import { BadRequestException, Body, Controller, Post, Req, UseGuards } from '@nestjs/common';
import { IsArray, IsDateString, IsNumber, IsObject, IsOptional, IsString } from 'class-validator';
import { PrismaService } from '../common/prisma.service';
import { Public, WorkerGuard } from '../common/security';

class TranscriptDto {
  @IsOptional()
  @IsString()
  externalTranscriptId?: string;

  @IsOptional()
  @IsString()
  sourceService?: string;

  @IsOptional()
  @IsString()
  eventId?: string;

  @IsOptional()
  @IsString()
  cameraId?: string;

  @IsOptional()
  @IsString()
  externalReceiptId?: string;

  @IsOptional()
  @IsString()
  saleSessionId?: string;

  @IsOptional()
  @IsString()
  audioUrl?: string;

  @IsDateString()
  startedAt!: string;

  @IsOptional()
  @IsDateString()
  finishedAt?: string;

  @IsOptional()
  @IsString()
  speaker?: string;

  @IsString()
  text!: string;

  @IsOptional()
  @IsString()
  language?: string;

  @IsOptional()
  @IsNumber()
  confidence?: number;

  @IsOptional()
  @IsArray()
  words?: any[];

  @IsOptional()
  @IsObject()
  metadata?: Record<string, any>;
}

@Public()
@UseGuards(WorkerGuard)
@Controller('workers/me/transcripts')
export class WorkerTranscriptsController {
  constructor(private db: PrismaService) {}

  private async ingest(worker: any, d: TranscriptDto) {
    const event = d.eventId
      ? await this.db.analytics_events.findFirst({ where: { id: BigInt(d.eventId), organization_id: worker.organization_id } })
      : null;
    if (d.eventId && !event) throw new BadRequestException('event not found');

    const camera = d.cameraId
      ? await this.db.cameras.findFirst({ where: { id: BigInt(d.cameraId), stores: { organization_id: worker.organization_id } } })
      : event?.camera_id
        ? await this.db.cameras.findFirst({ where: { id: event.camera_id, stores: { organization_id: worker.organization_id } } })
        : null;
    if (d.cameraId && !camera) throw new BadRequestException('camera not found');

    const receipt = d.externalReceiptId
      ? await this.db.receipts.findUnique({ where: { organization_id_external_receipt_id: { organization_id: worker.organization_id, external_receipt_id: d.externalReceiptId } } })
      : event?.receipt_id
        ? await this.db.receipts.findUnique({ where: { id: event.receipt_id } })
        : null;
    if (d.externalReceiptId && !receipt) throw new BadRequestException('receipt not found');

    const saleSession = d.saleSessionId
      ? await this.db.sale_sessions.findFirst({ where: { id: BigInt(d.saleSessionId), organization_id: worker.organization_id } })
      : receipt
        ? await this.db.sale_sessions.findUnique({ where: { receipt_id: receipt.id } })
        : event?.sale_session_id
          ? await this.db.sale_sessions.findUnique({ where: { id: event.sale_session_id } })
          : null;
    if (d.saleSessionId && !saleSession) throw new BadRequestException('sale session not found');

    const storeId = event?.store_id ?? receipt?.store_id ?? saleSession?.store_id ?? camera?.store_id;
    const workplaceId = event?.workplace_id ?? receipt?.workplace_id ?? saleSession?.workplace_id ?? camera?.workplace_id;
    if (!storeId) throw new BadRequestException('store scope could not be resolved');

    const data = {
      event_id: event?.id,
      organization_id: worker.organization_id,
      store_id: storeId,
      workplace_id: workplaceId,
      camera_id: camera?.id ?? event?.camera_id,
      receipt_id: receipt?.id ?? event?.receipt_id,
      sale_session_id: saleSession?.id ?? event?.sale_session_id,
      external_transcript_id: d.externalTranscriptId,
      source_service: d.sourceService ?? 'python-worker',
      audio_url: d.audioUrl,
      started_at: new Date(d.startedAt),
      finished_at: d.finishedAt ? new Date(d.finishedAt) : null,
      speaker: d.speaker ?? 'UNKNOWN',
      text: d.text,
      language: d.language,
      confidence: d.confidence,
      words: d.words,
      metadata: d.metadata ?? {},
    };

    if (data.external_transcript_id) {
      return this.db.event_transcripts.upsert({
        where: {
          source_service_external_transcript_id: {
            source_service: data.source_service,
            external_transcript_id: data.external_transcript_id,
          },
        },
        update: data,
        create: data,
      });
    }
    return this.db.event_transcripts.create({ data });
  }

  @Post()
  one(@Req() req: any, @Body() d: TranscriptDto) {
    return this.ingest(req.worker, d);
  }

  @Post('batch')
  batch(@Req() req: any, @Body() d: TranscriptDto[]) {
    return Promise.all(d.map(x => this.ingest(req.worker, x)));
  }
}
