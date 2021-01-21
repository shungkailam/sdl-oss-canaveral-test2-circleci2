import { initSequelize } from '../rest/sql-api/baseApi';
import { createBuiltinScriptRuntimes } from '../rest/db-scripts/scriptRuntimeHelper';

const USAGE = `\nUsage: node createBuiltinScriptRuntimes.js <tenant id>\n`;
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenantId = process.argv[2];

  let sql = initSequelize();

  // now create builtin script runtimes
  await createBuiltinScriptRuntimes(sql, tenantId);

  sql.close();
}

main();
