import * as uuidv4 from 'uuid/v4';
import { randomAttribute, randomCount, randomIPObject } from '../common';

export function randomEdge(ctx: any, apiVersion: string) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const serialNumber = randomAttribute('sn');
  const { ipAddress, gateway, subnet } = randomIPObject();
  const edgeDevices = 0;
  const doc = {
    id,
    tenantId,
    name,
    description,
    serialNumber,
    ipAddress,
    gateway,
    subnet,
    edgeDevices,
  };
  if (apiVersion === 'v1') {
    const storageCapacity = 0;
    const storageUsage = 0;
    return {
      storageCapacity,
      storageUsage,
      ...doc,
    };
  } else {
    return doc;
  }
}

// note: can't update serial number
export function randomEdgeUpdate(ctx: any, apiVersion: string, entity) {
  const updated = randomEdge(ctx, apiVersion);
  const { id, serialNumber } = entity;
  return { ...updated, id, serialNumber };
}

export function purifyEdge(edge: any, apiVersion: string) {
  const {
    id,
    tenantId,
    name,
    description,
    serialNumber,
    ipAddress,
    gateway,
    subnet,
    edgeDevices,
    storageCapacity,
    storageUsage,
  } = edge;
  const doc = {
    id,
    tenantId,
    name,
    description,
    serialNumber,
    ipAddress,
    gateway,
    subnet,
    edgeDevices,
  };
  if (apiVersion === 'v1') {
    return {
      storageCapacity,
      storageUsage,
      ...doc,
    };
  }
  return doc;
}
