import { initSequelize } from '../rest/sql-api/baseApi';
import { createAxios, loginToEdge } from '../generator/api';

const USAGE = `\nUsage: node loginAsEdge.js <cloudmgmt url> <edge id>\n`;

async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  const url = process.argv[2];
  const edgeId = process.argv[3];

  let sql = initSequelize();

  const ax = createAxios(url, null);
  const resp = await loginToEdge(ax, 'v1', edgeId);
  console.log('login response:', resp);

  sql.close();
}

main();
