import { initSequelize } from '../rest/sql-api/baseApi';
import { getAllTenantIDs, createDefaultProject } from './common';

const USAGE = `
Usage: node createDefaultProjectForAllTenants.js go

`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantIDs = await getAllTenantIDs(sql);

  console.log('Create default project for tenant ids: ', tenantIDs);

  await Promise.all(tenantIDs.map(id => createDefaultProject(sql, id, false)));

  sql.close();
}

main();
