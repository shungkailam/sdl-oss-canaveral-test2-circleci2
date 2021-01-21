import * as uuidv4 from 'uuid/v4';
import { randomAttribute } from '../common';

export function randomEdgeCert(ctx: any, apiVersion: string) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  return {
    id,
    tenantId,
    name,
  };
}
