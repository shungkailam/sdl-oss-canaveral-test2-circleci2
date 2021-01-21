import { generate } from './context';
import { getDBService } from '../rest/db-configurator/dbConfigurator';
import { DocType } from '../rest/model/baseModel';
import { encryptUserPassword } from '../rest/util/cryptoUtil';
import { createTenantRootCA } from '../getCerts/getCerts';
import {
  createAxiosForUser,
  testRBACUsersRead,
  testRBACEdgesRead,
  createAllEntities,
} from './read';
import { testRBACUsersCUD, testRBACEdgesCUD } from './cud';
import { sleep, entityCreationList } from './common';
import { initSequelize } from '../rest/sql-api/baseApi';
import { deleteTenant } from '../rest/db-scripts/common';

function logContext(ctx: any) {
  console.log('Context {');
  entityCreationList.forEach(e => {
    console.log(`>>> ${e.ctxKey} count: ${ctx[e.ctxKey].length}`);
  });
  console.log('Context }');
}

export async function generateAndTestRBAC(
  url: string,
  tenantId: string,
  cfsslUrl: string,
  apiVersion: string
) {
  const startTime = Date.now();
  try {
    const sql = initSequelize();

    await deleteTenant(sql, tenantId);

    // generate context
    const ctx = await generate(tenantId, apiVersion);

    logContext(ctx);

    // create tenant
    await getDBService().createTenant('', ctx.tenant);

    if (cfsslUrl) {
      cfsslUrl = cfsslUrl + '/rootca';
    }

    await createTenantRootCA(tenantId, cfsslUrl);

    // create super user
    const superUser = ctx.users[0];
    console.log(
      `super user email: ${superUser.email}, password: ${superUser.password}`
    );
    // - make a copy to encrypt password
    const su2 = { ...superUser };
    encryptUserPassword(su2);
    await getDBService().createDocument(tenantId, DocType.User, su2);

    console.log(`### tenant created: ${Date.now() - startTime}ms`);

    // login as super user
    const ax = await createAxiosForUser(url, superUser, apiVersion);

    console.log(`### logged in as super user: ${Date.now() - startTime}ms`);

    // create all entities through cloudmgmt using REST API
    await createAllEntities(ctx, ax, apiVersion);

    console.log(`### all entities created: ${Date.now() - startTime}ms`);

    await testRBACUsersRead(url, ctx, apiVersion);

    console.log(`### users read test done: ${Date.now() - startTime}ms`);

    await sleep(null, 1000);

    await testRBACEdgesRead(url, ctx, apiVersion);

    console.log(`### edges read test done: ${Date.now() - startTime}ms`);

    await testRBACUsersCUD(url, ctx, apiVersion);

    console.log(`### users CUD tests done: ${Date.now() - startTime}ms`);

    await testRBACEdgesCUD(url, ctx, apiVersion);

    // test successful, delete the tenant
    await deleteTenant(sql, tenantId);

    console.log(`### edges CUD tests done: ${Date.now() - startTime}ms`);
  } catch (e) {
    throw e;
  }
  console.log(`### generateAndTestRBAC done in ${Date.now() - startTime}ms`);
}
