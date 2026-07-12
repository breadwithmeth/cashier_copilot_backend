import { PrismaClient } from '@prisma/client';
import * as argon2 from 'argon2';

const db = new PrismaClient();

async function main() {
  const organization = await db.organizations.upsert({
    where: { code: 'DEMO' },
    update: {},
    create: { name: 'Демо организация', code: 'DEMO', timezone: 'Asia/Almaty' },
  });

  const eventTypes = [
    { code: 'PRODUCT_SCANNED', name: 'Товар отсканирован', category: 'SALE_CONTROL', severity: 'INFO' },
    { code: 'PRODUCT_SCANNED_WITHOUT_CUSTOMER', name: 'Товар отсканирован без клиента', category: 'SALE_CONTROL', severity: 'WARNING' },
    { code: 'CUSTOMER_WITHOUT_RECEIPT', name: 'Клиент обслужен без чека', category: 'SALE_CONTROL', severity: 'CRITICAL' },
    { code: 'PRODUCT_GIVEN_WITHOUT_PAYMENT', name: 'Товар передан без оплаты', category: 'SALE_CONTROL', severity: 'CRITICAL' },
    { code: 'RECEIPT_WITHOUT_CUSTOMER', name: 'Чек без клиента в кадре', category: 'SALE_CONTROL', severity: 'WARNING' },
    { code: 'RECEIVING_MISMATCH', name: 'Расхождение при приемке', category: 'RECEIVING', severity: 'WARNING' },
    { code: 'SERVICE_QUALITY', name: 'Проверка обслуживания', category: 'SERVICE', severity: 'INFO' },
  ];
  for (const eventType of eventTypes) {
    await db.event_types.upsert({
      where: { code: eventType.code },
      update: { name: eventType.name, category: eventType.category, default_severity: eventType.severity },
      create: {
        code: eventType.code,
        name: eventType.name,
        category: eventType.category,
        default_severity: eventType.severity,
        description: eventType.name,
      },
    });
  }

  const violationTypes = [
    { code: 'PRODUCT_SCANNED_WITHOUT_CUSTOMER', name: 'Товар отсканирован без клиента', risk: 'MEDIUM' },
    { code: 'CUSTOMER_WITHOUT_RECEIPT', name: 'Клиент обслужен без чека', risk: 'HIGH' },
    { code: 'PRODUCT_GIVEN_WITHOUT_PAYMENT', name: 'Товар передан без оплаты', risk: 'HIGH' },
    { code: 'RECEIPT_WITHOUT_CUSTOMER', name: 'Чек без клиента в кадре', risk: 'MEDIUM' },
    { code: 'RECEIVING_MISMATCH', name: 'Расхождение при приемке', risk: 'MEDIUM' },
  ];
  for (const violation of violationTypes) {
    const existing = await db.violation_types.findFirst({
      where: { organization_id: null, code: violation.code },
      select: { id: true },
    });
    if (existing) {
      await db.violation_types.update({
        where: { id: existing.id },
        data: { name: violation.name, risk_level: violation.risk, is_active: true },
      });
    } else {
      await db.violation_types.create({
        data: {
          organization_id: null,
          code: violation.code,
          name: violation.name,
          risk_level: violation.risk,
          visible_to_roles: ['MANAGER', 'ORGANIZATION_ADMIN', 'SUPER_ADMIN'],
        },
      });
    }
  }

  const email = process.env.BOOTSTRAP_ADMIN_USERNAME?.includes('@')
    ? process.env.BOOTSTRAP_ADMIN_USERNAME
    : `${process.env.BOOTSTRAP_ADMIN_USERNAME ?? 'admin'}@example.com`;
  await db.users.upsert({
    where: { email },
    update: {},
    create: {
      organization_id: organization.id,
      email,
      password_hash: await argon2.hash(process.env.BOOTSTRAP_ADMIN_PASSWORD ?? 'ChangeMe123!'),
      full_name: 'Администратор',
      role: 'ORGANIZATION_ADMIN',
    },
  });
}

main().finally(() => db.$disconnect());
