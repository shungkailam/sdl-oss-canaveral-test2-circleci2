import * as uuidv4 from 'uuid/v4';
import { randomAttribute, randomCount, pick, pickMany, range } from '../common';
import { DATA_TYPES } from '../../rest/model/baseModel';
import { randomCategoryInfo } from './category';

const DS_TYPES = ['Sensor', 'Gateway'];
const CONNECTIONS = ['Secure', 'Unsecure'];
const PROTOCOLS = ['MQTT', 'RTSP', 'GIGEVISION'];
const AUTH_TYPES_MAP = {
  MQTT: ['CERTIFICATE'],
  RTSP: ['PASSWORD'],
  GIGEVISION: ['TOKEN'],
};
const SENSOR_MODELS = ['Model 3', 'Model S', 'Model X'];

export function randomDataSourceFieldInfo(ctx: any) {
  const name = randomAttribute('name');
  const mqttTopic = randomAttribute('mqttTopic');
  const fieldType = pick(DATA_TYPES);
  return {
    name,
    mqttTopic,
    fieldType,
  };
}

export function randomDataSourceFieldInfoV2(ctx: any) {
  const name = randomAttribute('name');
  const topic = randomAttribute('topic');
  return {
    name,
    topic,
  };
}

export function randomDataSourceFieldSelector(
  ctx: any,
  fields: any[],
  fsCtx: any
) {
  let ok = false;
  let catInfo: any = null;

  while (!ok) {
    catInfo = randomCategoryInfo(ctx);
    catInfo.scope = [];
    const allKey = `${catInfo.id}`;
    const seenAll = fsCtx[allKey];
    if (!seenAll) {
      const all = !seenAll && Math.random() > 0.6;
      fsCtx[allKey] = true;
      const scope = all ? ['__ALL__'] : pickMany(fields).map(f => f.name);
      catInfo.scope = scope;
      ok = scope.length !== 0;
    }
  }
  catInfo.scope.sort();
  return catInfo;
}

function dsfsSortFn(a, b) {
  let c = a.id.localeCompare(b.id);
  if (c === 0) {
    c = a.value.localeCompare(b.value);
  }
  return c;
}

function nameSortFn(a, b) {
  return a.name.localeCompare(b.name);
}

export function randomDataSource(ctx: any, apiVersion: string, edge: any) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const { id: edgeId } = edge;

  const name = randomAttribute('name');
  const type = pick(DS_TYPES);

  const protocol = pick(PROTOCOLS);
  const authType = pick(AUTH_TYPES_MAP[protocol]);

  if (apiVersion === 'v1') {
    const sensorModel = pick(SENSOR_MODELS);
    const connection = pick(CONNECTIONS);

    const fields = range(randomCount(1, 5))
      .map(x => randomDataSourceFieldInfo(ctx))
      .sort(nameSortFn);
    const fsCtx: any = {};
    const selectors = range(randomCount(1, fields.length / 2))
      .map(x => randomDataSourceFieldSelector(ctx, fields, fsCtx))
      .sort(dsfsSortFn);
    selectors.forEach(selector => selector.scope.sort());

    return {
      id,
      tenantId,
      edgeId,
      name,
      type,
      protocol,
      authType,
      ifcInfo: null,
      sensorModel,
      connection,
      fields,
      selectors,
    };
  } else {
    const fields = range(randomCount(1, 5))
      .map(x => randomDataSourceFieldInfoV2(ctx))
      .sort(nameSortFn);
    const fsCtx: any = {};
    const selectors = range(randomCount(1, fields.length / 2))
      .map(x => randomDataSourceFieldSelector(ctx, fields, fsCtx))
      .sort(dsfsSortFn);
    selectors.forEach(selector => selector.scope.sort());

    return {
      id,
      tenantId,
      edgeId,
      name,
      type,
      protocol,
      authType,
      ifcInfo: null,
      fields,
      selectors,
    };
  }
}

export function randomDataSourceUpdate(
  ctx: any,
  apiVersion: string,
  edge: any,
  entity
) {
  const updated = randomDataSource(ctx, apiVersion, edge);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyDataSource(dataSource: any, apiVersion: string) {
  /* tslint:disable:no-unused-variable */
  const {
    sensorModel,
    connection,
    fields,
    selectors,
    version,
    createdAt,
    updatedAt,
    ...rest
  } = dataSource;

  if (selectors) {
    selectors.sort(dsfsSortFn);
    selectors.forEach(selector => selector.scope.sort());
  }

  if (fields) {
    fields.sort(nameSortFn);
  }

  if (apiVersion === 'v1') {
    return {
      ...rest,
      sensorModel,
      connection,
      fields,
      selectors,
    };
  } else {
    return {
      ...rest,
      fields,
      selectors,
    };
  }
}
