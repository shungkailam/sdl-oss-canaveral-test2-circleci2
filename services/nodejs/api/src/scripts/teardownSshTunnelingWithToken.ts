import { initSequelize } from '../rest/sql-api/baseApi';
import { createAxios } from '../generator/api';

const USAGE = `\nUsage: node teardownSshTunnelingWithToken.js <tenant id> <edge id> <api key> [<cloudmgmt url>]\n`;
async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  let exitCode = 0;
  const tenantId = process.argv[2];
  const edgeId = process.argv[3];
  const token = process.argv[4];
  const url = process.argv[5] || 'http://localhost:8080';

  let sql = initSequelize();

  try {
    const ax = createAxios(url, token);
    await ax.post('/v1/teardownsshtunneling', {
      tenantId,
      edgeId,
    });
    console.log('done');
  } catch (e) {
    // failed
    console.error(e);
    exitCode = 500;
  }

  sql.close();

  process.exit(exitCode);
}

main();
