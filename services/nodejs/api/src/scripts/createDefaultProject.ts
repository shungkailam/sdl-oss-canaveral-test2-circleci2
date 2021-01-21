import { initSequelize } from '../rest/sql-api/baseApi';
import { createDefaultProject } from './common';

const USAGE = `
Usage: node createDefaultProject.js <tenant id> [<force_update>]

If force_update is present, will execute update logic even if
the 'Default Project' already exists.
`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantId = process.argv[2];
  const forceUpdate = !!process.argv[3];

  await createDefaultProject(sql, tenantId, forceUpdate);

  sql.close();
}

main();
