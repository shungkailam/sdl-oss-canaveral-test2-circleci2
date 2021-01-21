const axios = require('axios');
import { dnsLookup } from '../rest/util/dnsUtil';
import { logger } from '../rest/util/logger';

export interface CertData {
  CACertificate: string;
  Certificate: string;
  PrivateKey: string;
}
export async function getCertData(tenantId: string, type: string) {
  // TODO FIXME - check tenantId
  let url;
  const host = await dnsLookup('cfsslserver-svc', 'localhost');
  const port = ':8888';
  const endpoint = '/certificates';
  const protocol = 'http://';
  url = protocol + host + port + endpoint;
  logger.info('getCertData: using url:', url);
  let response = await axios.post(
    url,
    {
      tenantId: tenantId,
      type: type,
    },
    { 'Content-Type': 'application/json' }
  );
  return response;
}
export async function getCerts(
  tenantId: string,
  type: string
): Promise<CertData> {
  const resp = await getCertData(tenantId, type);
  if (resp.status >= 200 && resp.status <= 299) {
    console.log(
      `>>> Certificate creation successful for tenantID: ${tenantId}`
    );
  } else {
    console.log(
      `>>> Certificate creation failed with error: ${resp.Status}, ${
        resp.statusText
      }`
    );
  }
  return resp.data;
}

export async function createTenantRootCAData(tenantId: string, url = '') {
  if (!url) {
    const host = await dnsLookup('cfsslserver-svc', 'localhost');
    const port = ':8888';
    const endpoint = '/rootca';
    const protocol = 'http://';
    url = protocol + host + port + endpoint;
  }
  logger.info('Create tenant root CA for tenantId: ', tenantId);
  let response = await axios.post(
    url,
    {
      tenantId: tenantId,
    },
    { 'Content-Type': 'application/json' }
  );
  return response;
}

export async function createTenantRootCA(tenantId: string, url = '') {
  const resp = await createTenantRootCAData(tenantId, url);
  if (resp.status >= 200 && resp.status <= 299) {
    console.log(
      `>>> Tenant root CA creation successful for tenantID: ${tenantId}`
    );
  } else {
    console.log(
      `>>> Tenant root CA creation failed with error: ${resp.Status}, ${
        resp.statusText
      }`
    );
  }
}
