import * as express from 'express';
import platformService from './rest/services/platform.service';
import { Exception } from './rest/model/baseModel';
import { logger } from './rest/util/logger';

class AuthFailedError implements Exception {
  public status = 401;
  public name = 'AuthFailedError';
  constructor(public message: string) {}
}

const EDGE_SCOPES = [
  'ctg.a',
  'cld.r',
  'agg',
  'ngg',
  'dsr.a',
  'dst.r',
  'edg.r',
  'prj.r',
  'scr.r',
  'sns.a',
  'usr.r',
  'log.w',
];
const CRUD = ['c', 'r', 'u', 'd'];
const EDGE_SCOPES_MAP = makeScopeMap(EDGE_SCOPES);
function makeScopeMap(scopes: string[]): any {
  const scopeMap: any = {};
  scopes.forEach(scope => {
    const tokens = scope.split('.');
    if (tokens.length === 1) {
      scopeMap[scope] = true;
    } else if (tokens.length === 2) {
      if (tokens[1] === 'a') {
        const token0 = tokens[0];
        CRUD.forEach(op => {
          scopeMap[`${token0}.${op}`] = true;
        });
      } else {
        scopeMap[scope] = true;
      }
    } else {
      logger.warn('Unexpected scope: ', scope);
    }
  });
  return scopeMap;
}

export function expressAuthentication(
  request: express.Request,
  securityName: string,
  scopes?: string[]
): Promise<any> {
  if (securityName === 'jwt') {
    const authHeader = <string>request.headers['authorization'] || '';
    logger.debug('REQ auth header:', authHeader);
    return new Promise(async (resolve, reject) => {
      const m = authHeader.match(/Bearer\s+(\S+)\s*/);
      if (!m) {
        reject(new AuthFailedError('No token provided'));
        return;
      }
      const token = m[1];
      try {
        const decoded = await platformService.getKeyService().jwtVerify(token);
        logger.debug('auth, found decoded token: ', decoded);
        let resolved = false;
        // Check if JWT contains all required scopes
        const { specialRole } = decoded;
        if (specialRole === 'admin') {
          resolved = true;
        } else if (specialRole === 'edge') {
          for (let scope of scopes) {
            const subScopes = scope.split(',');
            if (subScopes.every(sp => EDGE_SCOPES_MAP[sp])) {
              resolved = true;
              break;
            }
          }
        } else {
          for (let scope of scopes) {
            const subScopes = scope.split(',');
            if (subScopes.every(sp => decoded.scopes.includes(sp))) {
              resolved = true;
              break;
            }
          }
        }
        if (resolved) {
          resolve(decoded);
        } else {
          reject(new AuthFailedError('JWT does not contain required scope.'));
        }
      } catch (e) {
        logger.warn('auth caught exception:', e);
        reject(new AuthFailedError('Caught exception:' + e));
      }
    });
  }
}
