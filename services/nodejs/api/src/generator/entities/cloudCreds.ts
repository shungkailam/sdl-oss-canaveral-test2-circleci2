import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pick, wrapString } from '../common';

export function randomGCPCredential(ctx: any) {
  return [
    'type',
    'project_id',
    'private_key_id',
    'private_key',
    'client_email',
    'client_id',
    'auth_uri',
    'token_uri',
    'auth_provider_x509_cert_url',
    'client_x509_cert_url',
  ].reduce((acc, cur) => {
    const v = randomAttribute(cur);
    acc[cur] = v;
    return acc;
  }, {});
}

export function randomAWSCredential(ctx: any) {
  return ['accessKey', 'secret'].reduce((acc, cur) => {
    const v = randomAttribute(cur);
    acc[cur] = v;
    return acc;
  }, {});
}

const CLOUD_TYPES = ['AWS', 'GCP'];

export function randomCloudCreds(ctx: any, apiVersion: string) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const type = pick(CLOUD_TYPES);
  const awsCredential = type === 'AWS' ? randomAWSCredential(ctx) : null;
  const gcpCredential = type === 'GCP' ? randomGCPCredential(ctx) : null;
  const doc: any = {
    id,
    tenantId,
    name,
    description,
    type,
    awsCredential,
    gcpCredential,
  };
  return doc;
}

export function randomCloudCredsUpdate(ctx: any, apiVersion: string, entity) {
  const updated = randomCloudCreds(ctx, apiVersion);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyCloudCreds(cloudCreds: any, apiVersion: string) {
  const {
    id,
    tenantId,
    name,
    description,
    type,
    awsCredential,
    gcpCredential,
  } = cloudCreds;
  const doc: any = {
    id,
    tenantId,
    name,
    description,
    type,
    awsCredential,
    gcpCredential,
  };
  if (!awsCredential) {
    doc.awsCredential = null;
  } else {
    awsCredential.secret = wrapString(awsCredential.secret, '*', 0, 4);
  }
  if (!gcpCredential) {
    doc.gcpCredential = null;
  } else {
    gcpCredential.private_key = wrapString(
      gcpCredential.private_key,
      '*',
      0,
      4
    );
  }
  return doc;
}
