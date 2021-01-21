import * as jwt from 'jsonwebtoken';
import * as express from 'express';

export function getJwtToken(request: express.Request) {
  const authHeader = <string>request.headers['authorization'];
  if (!authHeader || authHeader.indexOf('Bearer ') !== 0) {
    throw new Error('No token provided');
  }
  const token = authHeader.substring('Bearer '.length);
  if (!token) {
    throw new Error('No token provided');
  }
  return jwt.decode(token);
}

export function getTenantId(request: express.Request): string {
  const jwtToken: any = getJwtToken(request);
  return jwtToken.tenantId;
}
