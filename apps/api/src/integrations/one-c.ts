import { BadRequestException, Body, Controller, Injectable, Post, Req, UnauthorizedException, UseGuards, CanActivate, ExecutionContext } from '@nestjs/common';
import { IsArray, IsDateString, IsNumber, IsObject, IsOptional, IsString } from 'class-validator';
import { timingSafeEqual } from 'crypto';
import { PrismaService } from '../common/prisma.service';
import { Public } from '../common/security';

class OneCBaseDto {
  @IsOptional()
  @IsString()
  organizationCode?: string;

  @IsString()
  storeCode!: string;

  @IsString()
  workplaceExternalId!: string;

  @IsDateString()
  occurredAt!: string;

  @IsOptional()
  @IsObject()
  payload?: Record<string, any>;
}

class OneCEventDto extends OneCBaseDto {
  @IsString()
  externalEventId!: string;

  @IsString()
  eventType!: string;

  @IsOptional()
  @IsString()
  externalReceiptId?: string;

  @IsOptional()
  @IsString()
  barcode?: string;

  @IsOptional()
  @IsString()
  productName?: string;

  @IsOptional()
  @IsNumber()
  quantity?: number;

  @IsOptional()
  @IsNumber()
  price?: number;

  @IsOptional()
  @IsString()
  currency?: string;
}

class OneCReceiptDto extends OneCBaseDto {
  @IsString()
  externalReceiptId!: string;

  @IsOptional()
  @IsString()
  externalOrderId?: string;

  @IsOptional()
  @IsString()
  cashierExternalId?: string;

  @IsOptional()
  @IsString()
  paymentMethod?: string;

  @IsOptional()
  @IsArray()
  items?: any[];

  @IsOptional()
  @IsObject()
  totals?: Record<string, any>;
}

@Injectable()
export class OneCGuard implements CanActivate {
  canActivate(ctx: ExecutionContext) {
    const configuredKey = process.env.ONE_C_API_KEY;
    if (!configuredKey) throw new UnauthorizedException('ONE_C_API_KEY is not configured');

    const req = ctx.switchToHttp().getRequest();
    const raw = req.headers['x-1c-key'] ?? req.headers['x-one-c-key'];
    const key = Array.isArray(raw) ? raw[0] : raw;
    if (typeof key !== 'string') throw new UnauthorizedException();

    const a = Buffer.from(key);
    const b = Buffer.from(configuredKey);
    if (a.length !== b.length || !timingSafeEqual(a, b)) throw new UnauthorizedException();
    return true;
  }
}

@Public()
@UseGuards(OneCGuard)
@Controller('integrations/1c')
export class OneCIntegrationController {
  constructor(private db: PrismaService) {}

  private paymentMethod(value?: string) {
    const raw = value?.trim().toUpperCase();
    if (!raw) return undefined;
    if (['CASH', 'CARD', 'BONUS', 'MIXED'].includes(raw)) return raw;
    if (['НАЛИЧНЫЕ', 'NAL', 'CASHLESS_CASH'].includes(raw)) return 'CASH';
    if (['КАРТА', 'CARD_PAYMENT', 'BANK_CARD'].includes(raw)) return 'CARD';
    if (['БОНУСЫ', 'BONUSES'].includes(raw)) return 'BONUS';
    throw new BadRequestException('paymentMethod must be CASH, CARD, BONUS or MIXED');
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

  private async resolveScope(d: OneCBaseDto) {
    const organizationCode = d.organizationCode ?? process.env.ONE_C_ORGANIZATION_CODE;
    if (!organizationCode) throw new BadRequestException('organizationCode is required');

    const organization = await this.db.organizations.findUnique({ where: { code: organizationCode } });
    if (!organization) throw new BadRequestException('organization not found');

    const store = await this.db.stores.findUnique({ where: { organization_id_code: { organization_id: organization.id, code: d.storeCode } } });
    if (!store) throw new BadRequestException('store not found');

    const workplace = await this.db.workplaces.findUnique({ where: { store_id_external_id: { store_id: store.id, external_id: d.workplaceExternalId } } });
    if (!workplace) throw new BadRequestException('workplace not found');

    return { organization, store, workplace };
  }

  @Post('events')
  async event(@Body() d: OneCEventDto, @Req() req: any) {
    const { organization, store, workplace } = await this.resolveScope(d);
    const payload = { ...(d.payload ?? {}), headers: { requestId: req.headers['x-request-id'] } };
    if (d.eventType === 'PRODUCT_SCANNED') {
      const barcode = d.barcode ?? d.payload?.barcode;
      if (!barcode || typeof barcode !== 'string') throw new BadRequestException('barcode is required for PRODUCT_SCANNED');
      const externalReceiptId = d.externalReceiptId ?? d.payload?.externalReceiptId;
      const receipt = externalReceiptId
        ? await this.db.receipts.findUnique({
          where: { organization_id_external_receipt_id: { organization_id: organization.id, external_receipt_id: externalReceiptId } },
          select: { id: true },
        })
        : null;
      const row = await this.db.product_scans.upsert({
        where: { organization_id_external_scan_id: { organization_id: organization.id, external_scan_id: d.externalEventId } },
        update: {
          store_id: store.id,
          workplace_id: workplace.id,
          external_receipt_id: externalReceiptId,
          receipt_id: receipt?.id,
          barcode,
          product_name: d.productName ?? d.payload?.productName ?? d.payload?.name,
          quantity: d.quantity ?? d.payload?.quantity,
          price: d.price ?? d.payload?.price,
          currency: d.currency ?? d.payload?.currency,
          occurred_at: new Date(d.occurredAt),
          payload,
        },
        create: {
          organization_id: organization.id,
          store_id: store.id,
          workplace_id: workplace.id,
          external_scan_id: d.externalEventId,
          external_receipt_id: externalReceiptId,
          receipt_id: receipt?.id,
          barcode,
          product_name: d.productName ?? d.payload?.productName ?? d.payload?.name,
          quantity: d.quantity ?? d.payload?.quantity,
          price: d.price ?? d.payload?.price,
          currency: d.currency ?? d.payload?.currency,
          occurred_at: new Date(d.occurredAt),
          payload,
        },
      });
      return this.json({ id: row.id, status: 'received', type: 'product_scan' });
    }

    const row = await this.db.external_events.upsert({
      where: { source_system_external_event_id: { source_system: '1C', external_event_id: d.externalEventId } },
      update: {
        event_type: d.eventType,
        organization_id: organization.id,
        store_id: store?.id,
        workplace_id: workplace?.id,
        occurred_at: new Date(d.occurredAt),
        payload,
        processing_status: 'received',
        processing_error: null,
        updated_at: new Date(),
      },
      create: {
        source_system: '1C',
        event_type: d.eventType,
        external_event_id: d.externalEventId,
        organization_id: organization.id,
        store_id: store?.id,
        workplace_id: workplace?.id,
        occurred_at: new Date(d.occurredAt),
        payload,
        processing_status: 'received',
      },
    });
    return this.json({ id: row.id, status: row.processing_status });
  }

  @Post('receipts')
  async receipt(@Body() d: OneCReceiptDto, @Req() req: any) {
    const { organization, store, workplace } = await this.resolveScope(d);
    const employee = d.cashierExternalId
      ? await this.db.employees.findFirst({ where: { organization_id: organization.id, external_id: d.cashierExternalId, is_active: true }, select: { id: true } })
      : null;
    const totals = d.totals ?? {};
    const paymentMethod = this.paymentMethod(d.paymentMethod ?? totals.paymentMethod);
    const payload = {
      externalReceiptId: d.externalReceiptId,
      externalOrderId: d.externalOrderId,
      cashierExternalId: d.cashierExternalId,
      paymentMethod,
      items: d.items ?? [],
      totals,
      data: d.payload ?? {},
      headers: { requestId: req.headers['x-request-id'] },
    };
    const occurredAt = new Date(d.occurredAt);
    const row = await this.db.$transaction(async (tx: any) => {
      const receipt = await tx.receipts.upsert({
        where: { organization_id_external_receipt_id: { organization_id: organization.id, external_receipt_id: d.externalReceiptId } },
        update: {
          store_id: store.id,
          workplace_id: workplace.id,
          employee_id: employee?.id,
          external_order_id: d.externalOrderId,
          cashier_external_id: d.cashierExternalId,
          operation_type: String(totals.operationType ?? d.payload?.operationType ?? 'SALE').toUpperCase(),
          receipt_status: String(totals.receiptStatus ?? d.payload?.receiptStatus ?? 'CLOSED').toUpperCase(),
          payment_method: paymentMethod,
          receipt_total: totals.receiptTotal ?? totals.amount,
          paid_amount: totals.paidAmount,
          change_amount: totals.changeAmount,
          bonus_amount: totals.bonusAmount,
          discount_amount: totals.discountAmount,
          occurred_at: occurredAt,
          printed_at: totals.printedAt ? new Date(totals.printedAt) : null,
          closed_at: totals.closedAt ? new Date(totals.closedAt) : occurredAt,
          items: d.items ?? [],
          totals,
          payload,
        },
        create: {
          organization_id: organization.id,
          store_id: store.id,
          workplace_id: workplace.id,
          employee_id: employee?.id,
          external_receipt_id: d.externalReceiptId,
          external_order_id: d.externalOrderId,
          cashier_external_id: d.cashierExternalId,
          operation_type: String(totals.operationType ?? d.payload?.operationType ?? 'SALE').toUpperCase(),
          receipt_status: String(totals.receiptStatus ?? d.payload?.receiptStatus ?? 'CLOSED').toUpperCase(),
          payment_method: paymentMethod,
          receipt_total: totals.receiptTotal ?? totals.amount,
          paid_amount: totals.paidAmount,
          change_amount: totals.changeAmount,
          bonus_amount: totals.bonusAmount,
          discount_amount: totals.discountAmount,
          occurred_at: occurredAt,
          printed_at: totals.printedAt ? new Date(totals.printedAt) : null,
          closed_at: totals.closedAt ? new Date(totals.closedAt) : occurredAt,
          items: d.items ?? [],
          totals,
          payload,
        },
      });

      await tx.receipt_items.deleteMany({ where: { receipt_id: receipt.id } });
      if (d.items?.length) {
        await tx.receipt_items.createMany({
          data: d.items.map((item: any, index: number) => ({
            receipt_id: receipt.id,
            organization_id: organization.id,
            store_id: store.id,
            workplace_id: workplace.id,
            line_number: Number(item.lineNumber ?? index + 1),
            external_product_id: item.externalProductId,
            barcode: item.barcode,
            product_name: String(item.productName ?? item.name ?? item.title ?? item.barcode ?? `item-${index + 1}`),
            quantity: item.quantity ?? 1,
            price: item.price ?? 0,
            line_total: item.lineTotal ?? item.amount ?? item.total ?? item.price ?? 0,
            discount_amount: item.discountAmount,
            is_container: Boolean(item.isContainer ?? item.container),
            container_type: item.containerType,
            payload: item,
          })),
        });
      }

      await tx.sale_sessions.upsert({
        where: { receipt_id: receipt.id },
        update: {
          organization_id: organization.id,
          store_id: store.id,
          workplace_id: workplace.id,
          employee_id: employee?.id,
          operation_type: receipt.operation_type,
          started_at: occurredAt,
          finished_at: receipt.closed_at ?? occurredAt,
          status: 'CLOSED',
        },
        create: {
          organization_id: organization.id,
          store_id: store.id,
          workplace_id: workplace.id,
          receipt_id: receipt.id,
          employee_id: employee?.id,
          operation_type: receipt.operation_type,
          started_at: occurredAt,
          finished_at: receipt.closed_at ?? occurredAt,
          status: 'CLOSED',
          metadata: {},
        },
      });

      return receipt;
    });

    const linkedAnalyticsEvents = await this.db.analytics_events.updateMany({
      where: {
        organization_id: organization.id,
        OR: [
          { external_receipt_id: d.externalReceiptId },
          ...(d.externalOrderId ? [{ external_order_id: d.externalOrderId }] : []),
        ],
      },
      data: {
        receipt_id: row.id,
        external_receipt_id: d.externalReceiptId,
        ...(d.externalOrderId ? { external_order_id: d.externalOrderId } : {}),
      },
    });
    await this.db.product_scans.updateMany({
      where: {
        organization_id: organization.id,
        external_receipt_id: d.externalReceiptId,
      },
      data: { receipt_id: row.id },
    });

    return this.json({ id: row.id, status: 'received', type: 'receipt', linkedAnalyticsEvents: linkedAnalyticsEvents.count });
  }
}
