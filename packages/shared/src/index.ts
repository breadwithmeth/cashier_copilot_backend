export const ROLES = ['SUPER_ADMIN','ORGANIZATION_ADMIN','MANAGER','OPERATOR','TECHNICIAN','VIEWER'] as const;
export type Role = typeof ROLES[number];
export interface AuthUser { sub:string; email:string; role:Role; organizationId:string|null }
export interface Page<T> { data:T[]; total:number; page:number; pageSize:number }
export const EVENT_STATUSES = ['NEW','IN_REVIEW','CONFIRMED','FALSE_POSITIVE','RESOLVED'] as const;
