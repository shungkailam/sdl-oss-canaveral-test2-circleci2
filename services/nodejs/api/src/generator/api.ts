import AxiosLib from 'axios';

import { getDBService } from '../rest/db-configurator/dbConfigurator';
import * as crypto2 from 'crypto2';
import { getPublicKeyFromCertificate } from '../rest/util/cryptoUtil';
import { DocType } from '../rest/model/baseModel';
import { EdgeCert } from '../rest/model/edgeCert';
import { Tenant } from '../rest/model/tenant';
import platformService from '../rest/services/platform.service';

export function createAxios(url, token) {
  const config = {
    baseURL: url,
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: 60000,
  };
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }
  return AxiosLib.create(config);
}

export async function getEntities(
  ax,
  apiVersion: string,
  entityType: string,
  entityId: string
) {
  try {
    const sfx = entityId ? `/${entityId}` : '';
    const entityPath = getMappedEntityPath(apiVersion, entityType, false);
    const path = `/${apiVersion}/${entityPath}${sfx}`;
    const resp = await ax.get(path);
    if (apiVersion === 'v1.0' && !entityId) {
      if (resp.data.result) {
        return resp.data.result;
      }
    }
    return resp.data;
  } catch (e) {
    throw e;
  }
}
export async function deleteEntity(
  ax,
  apiVersion: string,
  entityType: string,
  entityID: string
) {
  try {
    const entityPath = getMappedEntityPath(apiVersion, entityType, false);
    const path = `/${apiVersion}/${entityPath}/${entityID}`;
    return await ax.delete(path);
  } catch (e) {
    throw e;
  }
}
export async function createEntity(
  ax,
  apiVersion: string,
  entityType: string,
  doc
): Promise<any> {
  try {
    const entityPath = getMappedEntityPath(apiVersion, entityType, true);
    const path = `/${apiVersion}/${entityPath}`;
    return await ax.post(path, doc);
  } catch (e) {
    throw e;
  }
}
export async function updateEntity(
  ax,
  apiVersion: string,
  entityType: string,
  doc
) {
  try {
    const entityPath = getMappedEntityPath(apiVersion, entityType, false);
    const path = `/${apiVersion}/${entityPath}/${doc.id}`;
    return await ax.put(path, doc);
  } catch (e) {
    throw e;
  }
}

export async function login(ax, apiVersion: string, email, password) {
  try {
    const resp = await ax.post(`/${apiVersion}/login`, {
      email,
      password,
    });
    console.log(`login as ${email} successful, token=`, resp.data.token);
    return resp.data;
  } catch (e) {
    console.error('*** Failed:', e);
    throw e;
  }
}

export async function loginToEdge(ax: any, apiVersion: string, edgeId: string) {
  try {
    // first get edge cert
    const edgeCert: EdgeCert = await getDBService().getEdgeCert(edgeId);

    if (edgeCert) {
      // get tenant
      const tenant = await getDBService().findOneDocument<Tenant>(
        '',
        { id: edgeCert.tenantId },
        DocType.Tenant
      );

      if (tenant) {
        // decrypt the private key
        if (
          edgeCert.edgePrivateKey.indexOf('-----BEGIN RSA PRIVATE KEY-----') ===
          -1
        ) {
          console.log('decrypting private key...');
          edgeCert.edgePrivateKey = await platformService
            .getKeyService()
            .tenantDecrypt(edgeCert.edgePrivateKey, tenant.token);
        }

        const email = `${edgeCert.tenantId}|${edgeCert.edgeId}`;
        const privateKey = edgeCert.edgePrivateKey;
        const signature = await crypto2.sign(email, privateKey);

        const publicKey = getPublicKeyFromCertificate(edgeCert.edgeCertificate);
        const isSignatureValid = await crypto2.verify(
          email,
          publicKey,
          signature
        );
        if (!isSignatureValid) {
          throw Error('invalid signature');
        }

        const resp = await login(ax, apiVersion, email, signature);

        return resp;
      } else {
        throw Error('failed to get tenant for edge id ' + edgeId);
      }
    } else {
      throw Error('failed to get edge cert for edge id ' + edgeId);
    }
  } catch (e) {
    console.error('*** Failed:', e); // e.response.data);
    throw e;
  }
}

function getMappedEntityPath(
  apiVersion: string,
  entityType: string,
  create: boolean
): string {
  if (apiVersion === 'v1.0') {
    switch (entityType) {
      case 'application':
        return 'applications';
      case 'scripts':
        return 'functions';
      case 'scriptruntimes':
        return 'runtimeenvironments';
      case 'cloudcreds':
        return 'cloudprofiles';
      case 'datastreams':
        return 'datapipelines';
      case 'dockerprofiles':
        return 'containerregistries';
      default:
        break;
    }
  }
  return entityType;
}
