import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import platformService from '../rest/services/platform.service';
import { createBuiltinScriptRuntimes } from '../rest/db-scripts/scriptRuntimeHelper';
import { createDataTypeCategory } from '../rest/db-scripts/categoryHelper';
import { createDefaultProject } from '../scripts/common';
import { createTenantRootCA } from '../getCerts/getCerts';

const USAGE = `\nUsage: node createTenant.js <tenant id> <tenant name>\n`;
async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenant = {
    id: process.argv[2],
    name: process.argv[3],
    token: await platformService.getKeyService().genTenantToken(),
    version: 0,
    description: '',
  };

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  const doc = await getDBService().createTenant('', tenant);
  console.log('create tenant returns:', doc);

  // Create a root CA for this tenant
  await createTenantRootCA(tenant.id);

  // create builtin script runtimes
  await createBuiltinScriptRuntimes(sql, tenant.id);

  // create data type category
  await createDataTypeCategory(sql, tenant.id);

  // create default project
  await createDefaultProject(sql, tenant.id, false);

  if (sql) {
    sql.close();
  }
}

main();
