import { initSequelize } from '../rest/sql-api/baseApi';
import { createDataTypeCategory } from '../rest/db-scripts/categoryHelper';

const USAGE = `\nUsage: node createDataTypeCategory.js <tenant id>\n`;
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenantId = process.argv[2];

  let sql = initSequelize();

  // now create datatype category
  await createDataTypeCategory(sql, tenantId);

  sql.close();
}

main();
