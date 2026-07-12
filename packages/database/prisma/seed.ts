import { PrismaClient } from '@prisma/client';
import * as argon2 from 'argon2';

const db = new PrismaClient();

async function main() {
  const organization = await db.organizations.upsert({
    where: { code: 'DEMO' },
    update: {},
    create: { name: 'Демо организация', code: 'DEMO', timezone: 'Asia/Almaty' },
  });
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
