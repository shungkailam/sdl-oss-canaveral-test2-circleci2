import { initSequelize } from '../rest/sql-api/baseApi';
import { assignInfraAdminRoleToAllUsers } from './common';

const USAGE = `
Usage: node assignInfraAdminRoleToAllUsers.js <tenant id>

`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantId = process.argv[2];

  await assignInfraAdminRoleToAllUsers(sql, tenantId);

  sql.close();
}

main();
