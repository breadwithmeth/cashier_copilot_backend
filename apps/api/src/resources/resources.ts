import { BadRequestException, Body, Controller, Delete, Get, Param, Post, Put } from '@nestjs/common';
import type { AuthUser } from '@cashier/shared';
import { CurrentUser, Roles, maskStreamUrl } from '../common/security';
import { PrismaService } from '../common/prisma.service';

const map: Record<string, string> = {
  stores: 'stores',
  workplaces: 'workplaces',
  cameras: 'cameras',
  streams: 'camera_streams',
  'product-scans': 'product_scans',
  receipts: 'receipts',
  'receipt-items': 'receipt_items',
  'sale-sessions': 'sale_sessions',
  observations: 'video_observations',
  transcripts: 'event_transcripts',
  'service-checks': 'service_check_results',
  'violation-types': 'violation_types',
  'integration-errors': 'integration_errors',
  receivings: 'receivings',
  'receiving-items': 'receiving_items',
  workers: 'analytics_workers',
  rois: 'camera_rois',
  rules: 'rules',
  models: 'models',
  shifts: 'shifts',
  users: 'users',
};

@Controller('resources')
export class ResourcesController {
  constructor(private db: PrismaService) {}

  private repo(name: string) {
    if (!map[name]) throw new BadRequestException('unknown resource');
    return (this.db as any)[map[name]];
  }

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

  private workplaceWhere(user: AuthUser) {
    const orgId = this.orgId(user);
    return user.role === 'SUPER_ADMIN' ? {} : orgId ? { stores: { organization_id: orgId } } : { id: BigInt(-1) };
  }

  private cameraChildWhere(user: AuthUser) {
    const orgId = this.orgId(user);
    return user.role === 'SUPER_ADMIN' ? {} : orgId ? { cameras: { stores: { organization_id: orgId } } } : { id: BigInt(-1) };
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

  private normalize(name: string, data: any) {
    const value = { ...data };
    for (const key of ['organization_id', 'store_id', 'workplace_id', 'camera_id', 'roi_type_id', 'worker_id', 'model_version_id']) {
      if (value[key] !== undefined && value[key] !== null && value[key] !== '') value[key] = BigInt(value[key]);
      else delete value[key];
    }
    if (name === 'streams' && value.stream_url) value.stream_url = String(value.stream_url).trim();
    return value;
  }

  private validate(name: string, data: any) {
    if (name === 'stores' && (!data.name || !data.code)) {
      throw new BadRequestException('store name and code are required');
    }
    if (name === 'workplaces' && (!data.store_id || !data.name || !data.external_id)) {
      throw new BadRequestException('workplace store_id, name and external_id are required');
    }
    if (name === 'cameras' && (!data.workplace_id || !data.name || !data.code)) {
      throw new BadRequestException('camera workplace_id, name and code are required');
    }
    if (name === 'streams' && (!data.camera_id || !data.stream_type || !data.stream_url)) {
      throw new BadRequestException('stream camera_id, stream_type and stream_url are required');
    }
  }

  private writeError(error: any) {
    if (error?.code === 'P2002') throw new BadRequestException('resource with the same unique fields already exists');
    if (error?.code === 'P2003') throw new BadRequestException('related resource does not exist');
    if (error?.code === 'P2025') throw new BadRequestException('resource not found');
    throw error;
  }

  @Get(':name')
  async list(@CurrentUser() user: AuthUser, @Param('name') name: string) {
    let where: any = {};
    if (['stores', 'rules', 'users'].includes(name)) where = this.orgWhere(user);
    if (name === 'workplaces') where = this.workplaceWhere(user);
    if (name === 'cameras') where = this.cameraWhere(user);
    if (['product-scans', 'receipts', 'receipt-items', 'sale-sessions', 'observations', 'transcripts', 'service-checks', 'violation-types', 'integration-errors', 'receivings', 'receiving-items', 'workers'].includes(name)) where = this.orgWhere(user);
    if (['streams', 'rois'].includes(name)) where = this.cameraChildWhere(user);

    const rows = await this.repo(name).findMany({ where, take: 500 });
    const data = name === 'streams'
      ? rows.map((row: any) => ({ ...row, stream_url: row.stream_url && maskStreamUrl(row.stream_url) }))
      : rows;
    return this.json(data);
  }

  @Roles('SUPER_ADMIN', 'ORGANIZATION_ADMIN', 'TECHNICIAN')
  @Post(':name')
  async create(@CurrentUser() user: AuthUser, @Param('name') name: string, @Body() data: any) {
    this.validate(name, data);
    if (['stores', 'rules', 'users'].includes(name) && user.role !== 'SUPER_ADMIN') {
      const orgId = this.orgId(user);
      if (!orgId) throw new BadRequestException('user has no organization');
      data.organization_id = orgId;
    }
    if (name === 'workplaces') {
      const store = await this.db.stores.findFirst({ where: { id: BigInt(data.store_id), ...this.orgWhere(user) }, select: { id: true } });
      if (!store) throw new BadRequestException('store not found');
    }
    if (name === 'cameras') {
      const workplace = await this.db.workplaces.findFirst({
        where: { id: BigInt(data.workplace_id), ...this.workplaceWhere(user) },
        select: { store_id: true },
      });
      if (!workplace) throw new BadRequestException('workplace not found');
      data.store_id = workplace.store_id;
    }
    if (name === 'streams') {
      const camera = await this.db.cameras.findFirst({ where: { id: BigInt(data.camera_id), ...this.cameraWhere(user) }, select: { id: true } });
      if (!camera) throw new BadRequestException('camera not found');
    }
    try {
      return this.json(await this.repo(name).create({ data: this.normalize(name, data) }));
    } catch (error) {
      this.writeError(error);
    }
  }

  @Roles('SUPER_ADMIN', 'ORGANIZATION_ADMIN', 'TECHNICIAN')
  @Put(':name/:id')
  async update(@CurrentUser() user: AuthUser, @Param('name') name: string, @Param('id') id: string, @Body() data: any) {
    delete data.id;
    delete data.organization_id;
    if (name === 'cameras') {
      delete data.store_id;
      if (data.workplace_id) {
        const workplace = await this.db.workplaces.findFirst({
          where: { id: BigInt(data.workplace_id), ...this.workplaceWhere(user) },
          select: { store_id: true },
        });
        if (!workplace) throw new BadRequestException('workplace not found');
        data.store_id = workplace.store_id;
      }
    }
    try {
      return this.json(await this.repo(name).update({ where: { id: BigInt(id) }, data: this.normalize(name, data) }));
    } catch (error) {
      this.writeError(error);
    }
  }

  @Roles('SUPER_ADMIN', 'ORGANIZATION_ADMIN', 'TECHNICIAN')
  @Delete(':name/:id')
  async remove(@Param('name') name: string, @Param('id') id: string) {
    return this.json(await this.repo(name).update({ where: { id: BigInt(id) }, data: { is_active: false } }));
  }
}
