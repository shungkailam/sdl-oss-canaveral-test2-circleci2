import { initSequelize } from '../rest/sql-api/baseApi';
import { deleteTenant } from '../rest/db-scripts/common';

//
// Script to FULLY delete a tenant (including all its associated objects)
//
const USAGE = `\nUsage: node deleteTenant.js <tenant id>\n`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantId = process.argv[2];

  await deleteTenant(sql, tenantId);

  sql.close();
}

main();
