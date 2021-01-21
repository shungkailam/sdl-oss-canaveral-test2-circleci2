import { initSequelize } from '../rest/sql-api/baseApi';
import { getAllTenantIDs, doQuery } from './common';
import { createTenantRootCA } from '../getCerts/getCerts';

const USAGE = `
Usage: node createRootCAForAllTenants.js go

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
    await Promise.all(tenantIDs.map(id => createRootCA(sql, id)));
  } catch (e) {
    console.log('Failed to create root CA, caught exception:', e);
  }

  sql.close();
}

main();

async function createRootCA(sql: any, tenantId: string) {
  console.log('Fetching tenant_rootca for tenant_id ', tenantId);
  const rootCA = await doQuery(
    sql,
    `SELECT * FROM tenant_rootca_model WHERE tenant_id = '${tenantId}'`
  );

  if (rootCA.length === 0) {
    await createTenantRootCA(tenantId);
  } else {
    console.log('root ca already exists for tenant id', tenantId);
  }
}
