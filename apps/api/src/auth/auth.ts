import { Body, Controller, Get, Post, Req, UnauthorizedException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { IsEmail, IsString } from 'class-validator';
import { Throttle } from '@nestjs/throttler';
import { PrismaService } from '../common/prisma.service';
import { CurrentUser, Public } from '../common/security';
import * as argon2 from 'argon2';
import { createHash, randomBytes } from 'crypto';
import type { AuthUser } from '@cashier/shared';

class LoginDto { @IsEmail() email!: string; @IsString() password!: string }
class TokenDto { @IsString() refreshToken!: string }

@Controller('auth')
export class AuthController {
  constructor(private db: PrismaService, private jwt: JwtService) {}
  private hash(value: string) { return createHash('sha256').update(value).digest('hex') }
  private secret() { return process.env.JWT_ACCESS_SECRET ?? process.env.JWT_SECRET ?? 'development-only' }

  private async tokens(user: any, req: any) {
    const payload = { sub: user.id.toString(), email: user.email, role: user.role, organizationId: user.organization_id?.toString() ?? null };
    const accessToken = this.jwt.sign(payload, { secret: this.secret(), expiresIn: (process.env.ACCESS_TOKEN_TTL ?? '15m') as any });
    const refreshToken = randomBytes(48).toString('base64url');
    await this.db.refresh_tokens.create({ data: {
      user_id: user.id, token_hash: this.hash(refreshToken),
      expires_at: new Date(Date.now() + Number(process.env.REFRESH_TOKEN_DAYS ?? 30) * 864e5),
      ip_address: req.ip, user_agent: req.headers['user-agent'],
    }});
    return { accessToken, refreshToken };
  }

  @Public() @Throttle({ default: { limit: 5, ttl: 60000 } }) @Post('login')
  async login(@Body() dto: LoginDto, @Req() req: any) {
    const user = await this.db.users.findUnique({ where: { email: dto.email.toLowerCase() } });
    if (!user?.is_active || !(await argon2.verify(user.password_hash, dto.password))) throw new UnauthorizedException('Неверный логин или пароль');
    await this.db.users.update({ where: { id: user.id }, data: { last_login_at: new Date() } });
    return this.tokens(user, req);
  }

  @Public() @Post('refresh')
  async refresh(@Body() dto: TokenDto, @Req() req: any) {
    const token = await this.db.refresh_tokens.findFirst({ where: { token_hash: this.hash(dto.refreshToken), revoked_at: null, expires_at: { gt: new Date() } }, include: { users: true } });
    if (!token || !token.users.is_active) throw new UnauthorizedException();
    await this.db.refresh_tokens.update({ where: { id: token.id }, data: { revoked_at: new Date() } });
    return this.tokens(token.users, req);
  }

  @Public() @Post('logout') async logout(@Body() dto: TokenDto) {
    await this.db.refresh_tokens.updateMany({ where: { token_hash: this.hash(dto.refreshToken) }, data: { revoked_at: new Date() } });
    return { ok: true };
  }
  @Get('me') me(@CurrentUser() user: AuthUser) { return user }
}
