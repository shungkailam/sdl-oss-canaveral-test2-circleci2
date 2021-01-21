import { initSequelize } from '../rest/sql-api/baseApi';
import { getAllTenantIDs } from './common';
import { createBuiltinScriptRuntimes } from '../rest/db-scripts/scriptRuntimeHelper';

const USAGE = `
Usage: node createBuiltinScriptRuntimesForAllTenants.js go

`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantIDs = await getAllTenantIDs(sql);

  console.log('Create builtin script runtimes for tenant ids: ', tenantIDs);

  await Promise.all(tenantIDs.map(id => createBuiltinScriptRuntimes(sql, id)));

  sql.close();
}

main();
