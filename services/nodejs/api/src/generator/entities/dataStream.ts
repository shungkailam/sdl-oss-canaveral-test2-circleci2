import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pick, randomCount, range, pickMany } from '../common';
import { DATA_TYPES } from '../../rest/model/baseModel';
import {
  AWS_REGIONS,
  GCP_REGIONS,
  DataStreamDestination,
  DataStreamDestinations,
  EdgeStreamTypes,
  AWSStreamTypes,
  GCPStreamTypes,
} from '../../rest/model/dataStream';
import { randomCategoryInfo, dedupeCategoryInfos } from './category';

const ORIGINS = ['Data Source', 'Data Stream'];

function ciSortFn(a, b) {
  let c = a.id.localeCompare(b.id);
  if (c === 0) {
    c = a.value.localeCompare(b.value);
  }
  return c;
}

export function randomDataStream(ctx: any, apiVersion: string, project: any) {
  const { tenantId, cloudCredss, scripts } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const dataType = pick(DATA_TYPES);
  const origin = pick(ORIGINS);
  const isFromStream = origin === 'Data Stream';
  const originSelectors = isFromStream
    ? null
    : dedupeCategoryInfos(
        range(randomCount(1, 5)).map(x => randomCategoryInfo(ctx))
      ).sort(ciSortFn);
  // must fill in originId later as dataStrams not yet ready
  const originId = null; // isFromStream ? pick(dataStreams).id : null;
  const destination = pick(DataStreamDestinations);
  const isEdge = destination === DataStreamDestination.Edge;
  const projectId = project.id;
  const ccid = pick(project.cloudCredentialIds);
  const cloudCreds = cloudCredss.find(c => c.id === ccid);
  const cloudType = cloudCreds.type;
  const cloudCredsId = isEdge ? null : cloudCreds.id;
  const isAWS = !isEdge && cloudType === 'AWS';
  const isGCP = !isEdge && cloudType === 'GCP';
  const awsCloudRegion = isAWS ? pick(AWS_REGIONS) : null;
  const gcpCloudRegion = isGCP ? pick(GCP_REGIONS) : null;
  const edgeStreamType = isEdge ? pick(EdgeStreamTypes) : null;
  const awsStreamType = isAWS ? pick(AWSStreamTypes) : null;
  const gcpStreamType = isGCP ? pick(GCPStreamTypes) : null;
  const size = randomCount(200000, 1000000);
  const enableSampling = Math.random() > 0.5;
  const samplingInterval = randomCount(1000, 10000);
  const dataRetention = null;
  const txs = scripts.filter(s => s.projectId === projectId);
  const transformationArgsList = pickMany(txs).map(x => ({
    transformationId: x.id,
    args: null,
  }));

  return {
    id,
    tenantId,
    name,
    description,
    dataType,
    originSelectors,
    originId,
    destination,
    cloudType,
    cloudCredsId,
    awsCloudRegion,
    gcpCloudRegion,
    edgeStreamType,
    awsStreamType,
    gcpStreamType,
    size,
    enableSampling,
    samplingInterval,
    dataRetention,
    projectId,
    transformationArgsList,
  };
}

export function randomDataStreamUpdate(
  ctx: any,
  apiVersion: string,
  project: any,
  entity
) {
  const updated = randomDataStream(ctx, apiVersion, project);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyDataStream(dataStream: any, apiVersion: string) {
  const {
    id,
    tenantId,
    name,
    description,
    dataType,
    originSelectors: os,
    originId,
    destination,
    cloudType,
    cloudCredsId,
    awsCloudRegion,
    gcpCloudRegion,
    edgeStreamType,
    awsStreamType,
    gcpStreamType,
    size,
    enableSampling,
    samplingInterval,
    dataRetention,
    projectId,
    transformationArgsList,
  } = dataStream;
  const originSelectors = os ? os.sort(ciSortFn) : null;
  const doc: any = {
    id,
    tenantId,
    name,
    description,
    dataType,
    originSelectors,
    originId,
    destination,
    cloudType,
    cloudCredsId,
    awsCloudRegion,
    gcpCloudRegion,
    edgeStreamType,
    awsStreamType,
    gcpStreamType,
    size,
    enableSampling,
    samplingInterval,
    dataRetention,
    projectId,
    transformationArgsList,
  };
  if (!originId) {
    doc.originId = null;
  }
  if (!awsCloudRegion) {
    doc.awsCloudRegion = null;
  }
  if (!gcpCloudRegion) {
    doc.gcpCloudRegion = null;
  }
  if (!awsStreamType) {
    doc.awsStreamType = null;
  }
  if (!gcpStreamType) {
    doc.gcpStreamType = null;
  }
  if (!edgeStreamType) {
    doc.edgeStreamType = null;
  }
  if (!cloudCredsId) {
    doc.cloudCredsId = null;
  }

  return doc;
}
