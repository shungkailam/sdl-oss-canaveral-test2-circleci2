import { getUserContext, getEdgeContext } from './context';
import { createAxiosForUser, createAxiosForEdge } from './read';
import {
  // EntityCreation,
  // entityCreationList,
  EntityVerification,
  entityVerificationList,
  pick,
  range,
  sleep,
} from './common';
import { createEntity, updateEntity, deleteEntity, getEntities } from './api';

// import { range, randomCount, pick } from './common';
// import { randomTenant } from './entities/tenant';
import { randomCategory, randomCategoryUpdate } from './entities/category';
import { randomEdge, randomEdgeUpdate } from './entities/edge';
import {
  randomDataSource,
  randomDataSourceUpdate,
} from './entities/dataSource';
import {
  randomCloudCreds,
  randomCloudCredsUpdate,
  purifyCloudCreds,
} from './entities/cloudCreds';
import {
  randomDockerProfile,
  randomDockerProfileUpdate,
  purifyDockerProfile,
} from './entities/dockerProfile';
// import { randomProjectUser, randomAdminUser } from './entities/user';
import { randomProject, randomProjectUpdate } from './entities/project';
import {
  randomApplication,
  randomApplicationUpdate,
} from './entities/application';
import {
  randomDataStream,
  randomDataStreamUpdate,
} from './entities/dataStream';
import { randomScript, randomScriptUpdate } from './entities/script';
import {
  randomScriptRuntime,
  randomScriptRuntimeUpdate,
} from './entities/scriptRuntime';
import { randomUser, randomUserUpdate } from './entities/user';
import * as equal from 'fast-deep-equal';
import { getSha256 } from '../rest/util/cryptoUtil';

// Create Update Delete support functions

export async function testRBACUsersCUD(url: string, ctx, apiVersion: string) {
  return await testRBACUsersCUDRecursive(
    url,
    ctx,
    apiVersion,
    ctx.users.slice()
  );
}
async function testRBACUsersCUDRecursive(
  url: string,
  ctx,
  apiVersion: string,
  users
) {
  if (users.length) {
    const user = users.shift();
    return testRBACUserCUD(url, ctx, apiVersion, user).then(x =>
      testRBACUsersCUDRecursive(url, ctx, apiVersion, users)
    );
  }
}
async function testRBACUserCUD(url: string, ctx, apiVersion: string, user) {
  const ax = await createAxiosForUser(url, user, apiVersion);
  const userCtx = getUserContext(ctx, user);
  await testRBACCUD(ax, ctx, userCtx, apiVersion);
}
export async function testRBACEdgesCUD(url: string, ctx, apiVersion: string) {
  return await testRBACEdgesCUDRecursive(
    url,
    ctx,
    apiVersion,
    ctx.edges.slice()
  );
}
async function testRBACEdgesCUDRecursive(
  url: string,
  ctx,
  apiVersion: string,
  edges
) {
  if (edges.length) {
    const edge = edges.shift();
    return testRBACEdgeCUD(url, ctx, apiVersion, edge).then(x =>
      testRBACEdgesCUDRecursive(url, ctx, apiVersion, edges)
    );
  }
}
async function testRBACEdgeCUD(url: string, ctx, apiVersion: string, edge) {
  const ax = await createAxiosForEdge(url, edge, apiVersion);
  const edgeCtx = getEdgeContext(ctx, edge);
  await testRBACCUD(ax, ctx, edgeCtx, apiVersion);
}

async function testRBACCUD(ax, gCtx, ctx, apiVersion: string) {
  return testRBACCUDRecursive(
    ax,
    gCtx,
    ctx,
    apiVersion,
    entityVerificationList.slice()
  );
}

function testRBACCUDRecursive(
  ax,
  gCtx,
  ctx,
  apiVersion: string,
  evList: EntityVerification[]
) {
  if (evList.length) {
    const entity = evList.shift();
    return testRBACCUDEntityVerification(
      ax,
      gCtx,
      ctx,
      apiVersion,
      entity
    ).then(x => {
      return testRBACCUDRecursive(ax, gCtx, ctx, apiVersion, evList);
    });
  }
}
async function testRBACCUDEntityVerification(
  ax,
  gCtx,
  ctx,
  apiVersion: string,
  ev: EntityVerification
) {
  // create a list of random entities
  const entities = randomEntities(apiVersion, gCtx, ctx, ev);
  if (entities.length) {
    return await Promise.all(
      entities.map(entity =>
        testRBACCUDEntity(ax, gCtx, ctx, apiVersion, entity, ev)
      )
    );
  }
}
async function testRBACCUDEntity(
  ax,
  gCtx,
  ctx,
  apiVersion: string,
  entity,
  ev: EntityVerification
) {
  // perform entity creation,
  // if expect success, also perform Update, followed by delete
  // function createEntity(ax, entityType: string, doc) {
  let path = ev.entity === 'applications' ? 'application' : ev.entity;
  const expectSuccess = canCUD(ctx, entity, ev);
  const { name } = ctx;
  try {
    const resp = await createEntity(ax, apiVersion, path, entity);
    if (expectSuccess) {
      // ok
      console.log(`>>> ${name} create ${ev.entity} success as expected`);
      entity.id = entity.id || resp.data._id || resp.data.id;
      let ex = null;

      try {
        let entityCreated = await getEntities(ax, apiVersion, path, entity.id);
        entityCreated = ev.purifyFn(entityCreated, apiVersion);
        if (ev.ctxKey === 'dockerProfiles') {
          entity = purifyDockerProfile(entity, apiVersion);
        } else if (
          ev.ctxKey === 'cloudCredss' &&
          ctx.name.indexOf('edge[') !== 0
        ) {
          entity = purifyCloudCreds(entity, apiVersion);
        } else if (ev.ctxKey === 'users') {
          let { password, ...rest } = entity;
          password = getSha256(password);
          // don't compare password, as password is masked in API response
          entity = rest;
        }
        if (!equal(entity, entityCreated)) {
          ex = Error(`*** ${name} create ${ev.entity} - output != input`);
          console.log(`*** ${name} create ${ev.entity} - input:`, entity);
          console.log(
            `*** ${name} create ${ev.entity} - output:`,
            entityCreated
          );
        } else {
          console.log(
            `>>> ${name} create ${ev.entity} - output == input as expected`
          );
        }
      } catch (e) {
        console.error(`*** ${name} create ${ev.entity} - fetch failed:`, e);
        ex = e;
      }

      let entityUpdate = null;
      if (!ex) {
        try {
          // wait a bit
          await sleep(null, 200);
          // TODO randomly update entity
          entityUpdate = randomEntityUpdate(gCtx, ctx, apiVersion, entity, ev);
          await updateEntity(ax, apiVersion, path, entityUpdate);
          console.log(`>>> ${name} update ${ev.entity} success as expected`);
        } catch (e) {
          // update failed
          console.error(
            `*** ${name} update ${ev.entity} unexpected failure:`,
            e
          );
          ex = e;
        }
      }

      if (!ex) {
        try {
          let entityUpdated = await getEntities(
            ax,
            apiVersion,
            path,
            entity.id
          );
          entityUpdated = ev.purifyFn(entityUpdated, apiVersion);
          if (ev.ctxKey === 'dockerProfiles') {
            entityUpdate = purifyDockerProfile(entityUpdate, apiVersion);
          } else if (
            ev.ctxKey === 'cloudCredss' &&
            ctx.name.indexOf('edge[') !== 0
          ) {
            entityUpdate = purifyCloudCreds(entityUpdate, apiVersion);
          } else if (ev.ctxKey === 'users') {
            let { password, ...rest } = entityUpdate;
            password = getSha256(password);
            // don't compare password, as password is masked in API response
            entityUpdate = rest;
          }
          if (!equal(entityUpdate, entityUpdated)) {
            ex = Error(`*** ${name} update ${ev.entity} - output != input`);
            console.log(
              `*** ${name} update ${ev.entity} - input:`,
              entityUpdate
            );
            console.log(
              `*** ${name} update ${ev.entity} - output:`,
              entityUpdated
            );
          }
        } catch (e) {
          console.error(`*** ${name} update ${ev.entity} - fetch failed:`, e);
          ex = e;
        }
      }

      // now delete
      // for debugging, let's not delete if update fail
      if (!ex) {
        try {
          // wait a bit
          await sleep(null, 200);
          await deleteEntity(ax, apiVersion, path, entity.id);
          console.log(`>>> ${name} delete ${ev.entity} success as expected`);
        } catch (e) {
          // delete failed
          console.error(
            `*** ${name} delete ${ev.entity} unexpected failure:`,
            e
          );
          ex = e;
        }
      }
      if (ex) {
        throw ex;
      }
    } else {
      // fail
      console.error(
        `*** ${name} create ${ev.entity} unexpected success:`,
        entity
      );
      throw Error(`*** ${name} create ${ev.entity} unexpected success`);
    }
  } catch (e) {
    if (expectSuccess) {
      // fail
      console.error(`*** ${name} create ${ev.entity} unexpected failure:`, e);
      throw e;
    } else {
      // ok
      console.log(`>>> ${name} create ${ev.entity} failed as expected.`);
    }
  }
}
/**
 * Whether a given Create/Update/Delete operation is allowed by RBAC.
 * This function reflects the currently implemented RBAC rules.
 * @param ctx
 * @param entity
 * @param ev
 */
function canCUD(ctx, entity, ev: EntityVerification): boolean {
  const entityType = ev.entity;
  const isProjScopedEntity =
    entityType === 'applications' ||
    entityType === 'datastreams' ||
    entityType === 'scripts' ||
    entityType === 'scriptruntimes';

  if (ctx.user) {
    const { userProjects: projects } = ctx;
    const isInfraAdmin = ctx.user.role === 'INFRA_ADMIN';
    if (isProjScopedEntity) {
      return projects.some(p => p.id === entity.projectId);
    }
    return isInfraAdmin;
  } else if (ctx.edge) {
    const { userProjects: projects } = ctx;
    if (isProjScopedEntity) {
      return projects.some(p => p.id === entity.projectId);
    }
    return false;
  }
  return false;
}
function randomEntities(apiVersion: string, gCtx, ctx, ev: EntityVerification) {
  const count = 3;
  switch (ev.entity) {
    case 'users':
      return range(count).map(x => randomUser(gCtx, apiVersion));
    case 'edges':
      return range(count).map(x => randomEdge(gCtx, apiVersion));
    case 'datasources':
      return range(count).map(x =>
        randomDataSource(gCtx, apiVersion, pick(gCtx.edges))
      );
    case 'categories':
      return range(count).map(x => randomCategory(gCtx, apiVersion));
    case 'cloudcreds':
    case 'cloudprofiles':
      return range(count).map(x => randomCloudCreds(gCtx, apiVersion));
    case 'dockerprofiles':
    case 'containerregistries':
      return range(count).map(x => randomDockerProfile(gCtx, apiVersion));
    case 'projects':
      return range(count).map(x => randomProject(gCtx, apiVersion));
    case 'scriptruntimes':
    case 'runtimeenvironments':
      return range(count).map(x =>
        randomScriptRuntime(gCtx, apiVersion, pick(gCtx.projects))
      );
    case 'scripts':
    case 'functions':
      return range(count).map(x =>
        randomScript(gCtx, apiVersion, pick(gCtx.projects))
      );
    case 'applications':
      return range(count).map(x =>
        randomApplication(gCtx, apiVersion, pick(gCtx.projects))
      );
    case 'datastreams':
    case 'datapipelines':
      return range(count).map(x =>
        randomDataStream(gCtx, apiVersion, pick(gCtx.projects))
      );

    default:
      return [];
  }
}
function getEntityProject(ctx, entity) {
  return ctx.projects.find(p => p.id === entity.projectId);
}
function getEntityEdge(ctx, entity) {
  return ctx.edges.find(e => e.id === entity.edgeId);
}
function randomEntityUpdate(
  gCtx,
  ctx,
  apiVersion: string,
  entity,
  ev: EntityVerification
) {
  switch (ev.entity) {
    case 'users':
      return randomUserUpdate(gCtx, apiVersion, entity);
    case 'edges':
      return randomEdgeUpdate(gCtx, apiVersion, entity);
    case 'datasources':
      return randomDataSourceUpdate(
        gCtx,
        apiVersion,
        getEntityEdge(gCtx, entity),
        entity
      );
    case 'categories':
      return randomCategoryUpdate(gCtx, apiVersion, entity);
    case 'cloudcreds':
      return randomCloudCredsUpdate(gCtx, apiVersion, entity);
    case 'dockerprofiles':
      return randomDockerProfileUpdate(gCtx, apiVersion, entity);
    case 'projects':
      return randomProjectUpdate(gCtx, apiVersion, entity);
    case 'scriptruntimes':
      return randomScriptRuntimeUpdate(
        gCtx,
        apiVersion,
        getEntityProject(gCtx, entity),
        entity
      );
    case 'scripts':
      return randomScriptUpdate(
        gCtx,
        apiVersion,
        getEntityProject(gCtx, entity),
        entity
      );
    case 'applications':
      return randomApplicationUpdate(
        gCtx,
        apiVersion,
        getEntityProject(gCtx, entity),
        entity
      );
    case 'datastreams':
      return randomDataStreamUpdate(
        gCtx,
        apiVersion,
        getEntityProject(gCtx, entity),
        entity
      );

    default:
      return entity;
  }
}
