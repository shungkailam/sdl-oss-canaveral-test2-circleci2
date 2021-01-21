import { initSequelize, getDocument } from '../rest/sql-api/baseApi';
import platformService from '../rest/services/platform.service';
import { getAllTenantIDs, doQuery } from './common';
import { getCerts } from '../getCerts/getCerts';
import { Tenant } from '../rest/model/tenant';
import { DocType } from '../rest/model/baseModel';
import { EdgeCert } from '../rest/model/edgeCert';

const USAGE = `
Usage: node createEdgeCertsForAllTenants.js go

`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantIDs = await getAllTenantIDs(sql);

  console.log('Create root CA certificate for tenant ids: ', tenantIDs);

  try {
    await Promise.all(
      tenantIDs.map(id => createEdgeCertsIfNotPresent(sql, id))
    );
  } catch (e) {
    console.log('Failed to create edge certs, caught exception:', e);
  }

  sql.close();
}

main();

async function createEdgeCertsIfNotPresent(sql: any, tenantId: string) {
  console.log('Fetching tenant_rootca for tenant_id ', tenantId);
  let edges: EdgeCert[] = await doQuery(
    sql,
    `SELECT * FROM edge_cert_model WHERE (tenant_id = '${tenantId}' AND edge_certificate IS NULL)`
  );
  return createEdgeCertsThrottledRecursive(edges, sql, tenantId);
}

function createEdgeCertsThrottledRecursive(
  edges: EdgeCert[],
  sql: any,
  tenantId: string
) {
  if (edges.length === 0) {
    console.log('Edge certs exists for all edges of tenant id', tenantId);
  } else {
    // update 16 edges at a time till done
    const batchSize = 16;
    if (edges.length <= batchSize) {
      return Promise.all(
        edges.map(edge => createEdgeCerts(sql, tenantId, edge))
      );
    } else {
      const batch = edges.slice(0, batchSize);
      edges = edges.slice(batchSize);
      return Promise.all(
        batch.map(edge => createEdgeCerts(sql, tenantId, edge))
      ).then(x => createEdgeCertsThrottledRecursive(edges, sql, tenantId));
    }
  }
}

async function createEdgeCerts(sql: any, tenantId: string, entry: EdgeCert) {
  const tenant = await getDocument<Tenant>(null, tenantId, DocType.Tenant);
  const edgeCert = await getCerts(tenantId, 'server');
  const edgePrivateKey = await platformService
    .getKeyService()
    .tenantEncrypt(edgeCert.PrivateKey, tenant.token);
  const mqttClientCert = await getCerts(tenantId, 'client');
  const mqttClientPrivateKey = await platformService
    .getKeyService()
    .tenantEncrypt(mqttClientCert.PrivateKey, tenant.token);
  console.log('entry id: ***************** ', entry.id);
  await sql.query(
    `UPDATE edge_cert_model SET edge_certificate = '${
      edgeCert.Certificate
    }', edge_private_key = '${edgePrivateKey}', client_certificate = '${
      mqttClientCert.Certificate
    }', client_private_key = '${mqttClientPrivateKey}' WHERE id = '${
      entry.id
    }'`,
    { type: sql.QueryTypes.INSERT }
  );
}
