import { randomAttribute } from '../common';
import platformService from '../../rest/services/platform.service';

export async function randomTenant(
  ctx: any,
  apiVersion: string,
  tenantId: string
) {
  const id = tenantId;
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const token = await platformService.getKeyService().genTenantToken();
  const version = 0;

  return {
    id,
    version,
    name,
    description,
    token,
  };
}
