import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';
import { deleteTenant } from '../rest/db-scripts/common';

//
// Script to FULLY delete all RBAC test tenants (including all its associated objects)
//
const USAGE = `\nUsage: node deleteAllRBACTestTenants.js go\n`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantIds = (await doQuery(
    sql,
    `SELECT id FROM tenant_model WHERE id LIKE 'tenant-id-rbac-test%'`
  )).map(t => t.id);
  console.log('got tenant ids:', tenantIds);
  await Promise.all(tenantIds.map(id => deleteTenant(sql, id)));

  sql.close();
}

main();
