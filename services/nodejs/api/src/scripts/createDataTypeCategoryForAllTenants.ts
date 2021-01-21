import { initSequelize } from '../rest/sql-api/baseApi';
import { getAllTenantIDs } from './common';
import { createDataTypeCategory } from '../rest/db-scripts/categoryHelper';

const USAGE = `
Usage: node createDataTypeCategoryForAllTenants.js go

`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantIDs = await getAllTenantIDs(sql);

  console.log('Create datatype category for tenant ids: ', tenantIDs);

  await Promise.all(tenantIDs.map(id => createDataTypeCategory(sql, id)));

  sql.close();
}

main();
