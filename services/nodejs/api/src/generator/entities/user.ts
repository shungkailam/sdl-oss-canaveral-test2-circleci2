import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pick } from '../common';

const roles = ['INFRA_ADMIN', 'USER'];

export function randomUser(ctx: any, apiVersion: string) {
  const role = pick(roles);
  return randomFixedRoleUser(ctx, apiVersion, role);
}

export function randomUserUpdate(ctx: any, apiVersion: string, entity) {
  const updated = randomUser(ctx, apiVersion);
  const { id } = entity;
  return { ...updated, id };
}

export function randomProjectUser(ctx: any, apiVersion: string) {
  return randomFixedRoleUser(ctx, apiVersion, 'USER');
}

export function randomAdminUser(ctx: any, apiVersion: string) {
  return randomFixedRoleUser(ctx, apiVersion, 'INFRA_ADMIN');
}
function randomFixedRoleUser(ctx: any, apiVersion: string, role: string) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const email = randomAttribute('email');
  const password = randomAttribute('P@ssw0rd');
  const version = 0;
  return {
    id,
    version,
    tenantId,
    name,
    email,
    password,
    role,
  };
}

export function purifyUser(user: any, apiVersion: string) {
  const { id, tenantId, name, email, password, role } = user;
  const version = 0;
  // don't compare password, as password is masked in API response
  return {
    id,
    version,
    tenantId,
    name,
    email,
    // password,
    role,
  };
}
