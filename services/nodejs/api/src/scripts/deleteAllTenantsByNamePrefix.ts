import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';
import { deleteTenant } from '../rest/db-scripts/common';

//
// Script to FULLY delete all tenants with given name prefix (including all its associated objects)
// (The minimum name prefix length allowed is 6.)
//
const USAGE = `\nUsage: node deleteAllTenantsByNamePrefix.js <name prefix> go\n`;

async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();
  const namePrefix = process.argv[2];

  // basic safe-guard, don't allow name prefix too short
  if (namePrefix.length < 6) {
    console.log(`Name prefix too short (<6): ${namePrefix}`);
    process.exit(1);
  }

  const tenantIds = (await doQuery(
    sql,
    `SELECT id FROM tenant_model WHERE name LIKE '${namePrefix}%'`
  )).map(t => t.id);
  console.log('got tenant ids:', tenantIds);
  await Promise.all(tenantIds.map(id => deleteTenant(sql, id)));

  sql.close();
}

main();
