import { initSequelize } from '../rest/sql-api/baseApi';
import { createAxios } from '../generator/api';

// TODO FIXME - can't get TS jwt token signing to inter-operate (e.g., using createAdminToken)
// with golang, so use email / password login for now...

// quiet version of login
async function login(ax, email, password) {
  const resp = await ax.post('/v1/login', {
    email,
    password,
  });
  return resp.data;
}

const USAGE = `\nUsage: node teardownSshTunneling.js <tenant id> <edge id> <email> <password> [<cloudmgmt url>]\n`;
async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  let exitCode = 0;
  const tenantId = process.argv[2];
  const edgeId = process.argv[3];
  const email = process.argv[4];
  const password = process.argv[5];
  const url = process.argv[6] || 'http://localhost:8080';

  let sql = initSequelize();

  try {
    const ax0 = createAxios(url, null);
    const { token } = await login(ax0, email, password);
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
