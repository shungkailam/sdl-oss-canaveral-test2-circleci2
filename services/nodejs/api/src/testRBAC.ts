import { initSequelize } from './rest/sql-api/baseApi';
import { generateAndTestRBAC } from './generator/generator';

const USAGE = `\nUsage: node testRBAC.js go [<cloudmgmt url> <tenant id> <cfssl url> <api version>]\n`;

async function main() {
  let ok = true;
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  const URL = process.argv[3] || 'https://test.ntnxsherlock.com';
  const tenantId = process.argv[4] || 'tenant-id-rbac-test';
  const cfsslURL = process.argv[5] || 'https://cfssl-test.ntnxsherlock.com';
  let apiVersion = process.argv[6];

  // randomly pick apiVersion if not specified
  // we can tweak this later, but for now we want both versions to be covered by tests
  if (!apiVersion) {
    apiVersion = Math.random() > 0.5 ? 'v1.0' : 'v1';
  }

  console.log('tenant id: ', tenantId);
  console.log('Test using API version: ', apiVersion);

  let sql = initSequelize();

  console.log('init sql done');

  try {
    await generateAndTestRBAC(URL, tenantId, cfsslURL, apiVersion);
  } catch (e) {
    console.log('caught exception:', e);
    ok = false;
  }
  sql.close();

  if (!ok) {
    process.exit(2);
  }
  process.exit(0);
}

main();
