import { DocType } from '../model/baseModel';
import { Tenant } from '../model/tenant';
import { TenantRootCA } from '../model/tenantRootca';
import { modelFromEs } from '../util/esUtil';
import * as omit from 'object.omit';
import { getDBService } from '../db-configurator/dbConfigurator';

export async function getAllTenants(): Promise<Tenant[]> {
  return getDBService().getAllDocuments<Tenant>(null, DocType.Tenant);
}

export async function getAllTenantRootCAs(): Promise<TenantRootCA[]> {
  return getDBService().getAllDocuments<TenantRootCA>(
    null,
    DocType.TenantRootCA
  );
}

/**
 * Get tenant with matching token.
 *
 * @param token
 * @returns {Promise<Tenant>}
 * @throws Error - if tenant not found or not unique or can't connect to ElasticSearch
 */
export async function getTenantWithToken(token): Promise<Tenant> {
  const tenants = await getAllTenants();
  if (tenants.length) {
    const tnts = tenants.filter(tenant => tenant.token === token);
    if (tnts.length === 1) {
      return tnts[0];
    } else if (tnts.length) {
      throw Error(
        'Ambiguous Exception: getTenantWithToken: more than one tenant matching the token ' +
          token
      );
    }
  }
  throw Error(
    'NotFoundException: getTenantWithToken: no tenant found with the token ' +
      token
  );
}
