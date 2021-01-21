import { initSequelize } from '../sql-api/baseApi';

import { TENANT_ID, TENANT_ID_2, getUser } from './dataDB';

import { DocType, DocTypes } from '../model/baseModel';
import { getDBService, isSQL } from '../db-configurator/dbConfigurator';
import platformService from '../services/platform.service';

const dbService = getDBService();
const { createAllTables, createDocument } = dbService;

// main function
// declare as async so we can use ES7 async/await
async function main() {
  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  // create all tables first
  // TODO FIXME: createTable is async on DynamoDB side, must wait for table to be ACTIVE
  await createAllTables();

  console.log('Creating new tenant records');
  await createTenantUser(TENANT_ID, 'waldot');
  await createTenantUser(TENANT_ID_2, 'Rocket Blue');

  setTimeout(async () => {
    if (isSQL()) {
      await sql.close();
    }
  }, 2000);
}

async function createTenantUser(tenantId, tenantName) {
  const tenantToken = await platformService.getKeyService().genTenantToken();
  // create tenant
  const doc = {
    id: tenantId,
    name: tenantName,
    token: tenantToken,
  };
  // create tenant
  await createDocument(tenantId, DocType.Tenant, doc);

  const user = getUser(tenantId);
  if (user) {
    await createDocument(tenantId, DocType.User, user);
  }
}

main();
