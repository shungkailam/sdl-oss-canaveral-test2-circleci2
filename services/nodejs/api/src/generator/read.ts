import { getUserContext, getEdgeContext } from './context';
import {
  createAxios,
  login,
  loginToEdge,
  createEntity,
  getEntities,
} from './api';
import {
  EntityCreation,
  entityCreationList,
  EntityVerification,
  entityVerificationList,
} from './common';

import { verifyEntities, sleep } from './common';
import * as equal from 'fast-deep-equal';
import { getSha256 } from '../rest/util/cryptoUtil';

export async function createAxiosForUser(
  url: string,
  user,
  apiVersion: string
) {
  const ax = createAxios(url, null);
  const resp = await login(ax, apiVersion, user.email, user.password);
  return createAxios(url, resp.token);
}
export async function createAxiosForEdge(
  url: string,
  edge,
  apiVersion: string
) {
  const ax = createAxios(url, null);
  const resp = await loginToEdge(ax, apiVersion, edge.id);
  return createAxios(url, resp.token);
}

function getSleepTime(key: string): number {
  if (key === 'projects') {
    return 2000;
  }
  return 1000;
}

// NOTE: Use recursion here since create must be done sequentially over entity types
function createAllEntitiesRecursive(
  ctx,
  ax,
  apiVersion: string,
  ecList: EntityCreation[]
) {
  if (ecList.length) {
    const entity = ecList.shift();
    let entities: any = null;
    if (entity.ctxKey === 'users') {
      // omit super user already created
      entities = ctx[entity.ctxKey].slice(1);
    } else {
      entities = ctx[entity.ctxKey].slice();
    }
    // return Promise.all(entities.map(c => createEntity(ax, entity.entity, c)))
    return createAllEntitiesThrottled(ax, apiVersion, entities, entity)
      .then(x => sleep(x, getSleepTime(entity.ctxKey)))
      .then(
        x => {
          console.log('... created ' + entity.ctxKey);
          return createAllEntitiesRecursive(ctx, ax, apiVersion, ecList);
        },
        e => {
          console.log('*** failed to create ' + entity.ctxKey, e);
          return Promise.reject(e);
        }
      );
  }
}

function createAllEntitiesThrottled(
  ax,
  apiVersion: string,
  entities,
  ec: EntityCreation
) {
  return createAllEntitiesThrottledRecursive(ax, apiVersion, entities, ec);
}
const batchSize = 4;
function createAllEntitiesThrottledRecursive(
  ax,
  apiVersion: string,
  entities,
  ec: EntityCreation
) {
  // do 4 at a time till done
  if (entities.length <= batchSize) {
    return Promise.all(
      entities.map(c => createEntity(ax, apiVersion, ec.entity, c))
    );
  } else {
    const batch = entities.slice(0, batchSize);
    entities = entities.slice(batchSize);
    return Promise.all(
      batch.map(c => createEntity(ax, apiVersion, ec.entity, c))
    ).then(x =>
      createAllEntitiesThrottledRecursive(ax, apiVersion, entities, ec)
    );
  }
}
// create all entities through cloudmgmt using REST API
export async function createAllEntities(ctx, ax, apiVersion: string) {
  const ecList = entityCreationList.slice();
  await createAllEntitiesRecursive(ctx, ax, apiVersion, ecList);
  console.log('done creating all entities');
}

async function verifyReadByID(
  ax,
  apiVersion: string,
  entity,
  entities,
  ev: EntityVerification,
  ctx
) {
  let targetEntity = entities.find(e => e.id === entity.id);
  if (targetEntity && ev.ctxKey === 'dockerProfiles') {
    // drop pwd, credentials, userName
    targetEntity = ev.purifyFn(targetEntity, apiVersion);
  } else if (targetEntity && ev.ctxKey === 'cloudCredss') {
    // mask password etc.
    if (ctx.name.indexOf('edge[') !== 0) {
      targetEntity = ev.purifyFn(targetEntity, apiVersion);
    }
  } else if (targetEntity && ev.ctxKey === 'users') {
    let { password, ...rest } = targetEntity;
    password = getSha256(password);
    // don't compare password, as password is masked in API response
    targetEntity = rest;
  }
  const path = ev.entity === 'applications' ? 'application' : ev.entity;

  try {
    const e = await getEntities(ax, apiVersion, path, entity.id);
    const pe = ev.purifyFn(e, apiVersion);
    if (targetEntity) {
      if (!equal(targetEntity, pe)) {
        const err = `*** get entity by id mismatch: ${ev.entity} id=${
          entity.id
        }`;
        console.error(err);
        console.error('*** entity:', pe);
        console.error('*** target entity:', targetEntity);
        throw Error(err);
      }
      console.log(
        `>>> get entity by id successful: ${ev.entity} id=${entity.id}`
      );
      return e;
    } else {
      const err = `*** get entity by id should fail: ${ev.entity} id=${
        entity.id
      }`;
      console.error(err);
      console.error('*** entity:', pe);
      console.error('*** target entity:', targetEntity);
      throw Error(err);
    }
  } catch (x) {
    if (targetEntity) {
      const err = `*** get entity by id should succeed: ${ev.entity} id=${
        entity.id
      }`;
      console.error(err);
      console.error('*** target entity:', targetEntity);
      throw Error(err);
    } else {
      // expected, so no op
      console.log(
        `>>> get entity by id failed as expected: ${ev.entity} id=${entity.id}`
      );
    }
  }
}

function testRBACReadByIDRecursive(
  ax,
  apiVersion: string,
  gEntities,
  entities,
  ev: EntityVerification,
  ctx
) {
  if (gEntities.length) {
    const entity = gEntities.shift();
    return verifyReadByID(ax, apiVersion, entity, entities, ev, ctx).then(x =>
      testRBACReadByIDRecursive(ax, apiVersion, gEntities, entities, ev, ctx)
    );
  }
}
function testRBACReadByID(
  ax,
  apiVersion: string,
  gCtx,
  ctx,
  entity: EntityVerification
) {
  const gEntities = gCtx[entity.ctxKey].slice();
  const entities = ctx[entity.ctxKey];
  return testRBACReadByIDRecursive(
    ax,
    apiVersion,
    gEntities,
    entities,
    entity,
    ctx
  );
}

function testRBACReadRecursive(
  ax,
  apiVersion: string,
  gCtx,
  ctx,
  evList: EntityVerification[]
) {
  if (evList.length) {
    const entity = evList.shift();
    return getEntities(ax, apiVersion, entity.entity, null)
      .then(entities => {
        return verifyEntities(
          ctx,
          apiVersion,
          entities,
          entity.ctxKey,
          entity.purifyFn
        );
      })
      .then(x => {
        return testRBACReadByID(ax, apiVersion, gCtx, ctx, entity);
      })
      .then(x => {
        return testRBACReadRecursive(ax, apiVersion, gCtx, ctx, evList);
      });
  }
}

async function testRBACRead(ax, apiVersion: string, gCtx, ctx) {
  return testRBACReadRecursive(
    ax,
    apiVersion,
    gCtx,
    ctx,
    entityVerificationList.slice()
  );
}

async function testRBACUserRead(url: string, ctx, apiVersion: string, user) {
  try {
    const ax = await createAxiosForUser(url, user, apiVersion);
    const userCtx = getUserContext(ctx, user);
    await testRBACRead(ax, apiVersion, ctx, userCtx);
  } catch (e) {
    console.log(
      `>>> testRBACUserRead - user: ${user.email} - caught exception:`,
      e
    );
    throw e;
  }
}
async function testRBACEdgeRead(url: string, ctx, apiVersion: string, edge) {
  try {
    console.log('testRBACEdgeRead: create axioss for edge');
    const ax = await createAxiosForEdge(url, edge, apiVersion);
    console.log('testRBACEdgeRead: get context for edge');
    const edgeCtx = getEdgeContext(ctx, edge);
    console.log('testRBACEdgeRead: checking RBAC read for edge');
    await testRBACRead(ax, apiVersion, ctx, edgeCtx);
  } catch (e) {
    console.log(
      `>>> testRBACEdgeRead - edge: ${edge.id} - caught exception:`,
      e
    );
    throw e;
  }
}

function testRBACUsersReadRecursive(
  url: string,
  ctx: any,
  apiVersion: string,
  users: any[]
) {
  if (users.length) {
    const user = users.shift();
    return testRBACUserRead(url, ctx, apiVersion, user)
      .then(x => sleep(x, 1000))
      .then(x => testRBACUsersReadRecursive(url, ctx, apiVersion, users));
  }
}

// test RBAC read for each user
// do this sequentially to reduce concurrent traffic to AWS
export async function testRBACUsersRead(
  url: string,
  ctx: any,
  apiVersion: string
) {
  const testAllUsers = false;
  let users: any[] = [];
  if (testAllUsers) {
    users = ctx.users.slice();
  } else {
    // one project user, one admin user
    const projUser = ctx.users[1];
    const adminUser = ctx.users[ctx.users.length - 1];
    users = [projUser, adminUser];
  }
  return await testRBACUsersReadRecursive(url, ctx, apiVersion, users);
}

function testRBACEdgesReadRecursive(
  url: string,
  ctx: any,
  apiVersion: string,
  edges: any[]
) {
  if (edges.length) {
    const edge = edges.shift();
    return testRBACEdgeRead(url, ctx, apiVersion, edge)
      .then(x => sleep(x, 1000))
      .then(x => testRBACEdgesReadRecursive(url, ctx, apiVersion, edges));
  }
}

// test RBAC read for each edge
// do this sequentially to reduce concurrent traffic to AWS
export async function testRBACEdgesRead(
  url: string,
  ctx: any,
  apiVersion: string
) {
  let testAllEdges = false;
  let edges: any[] = [];
  if (testAllEdges || ctx.edges.length <= 2) {
    edges = ctx.edges.slice();
  } else {
    edges = ctx.edges.slice(0, 2);
  }
  return await testRBACEdgesReadRecursive(url, ctx, apiVersion, edges);
}
