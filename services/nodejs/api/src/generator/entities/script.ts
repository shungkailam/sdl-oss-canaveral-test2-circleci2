import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pick } from '../common';

const SCRIPT_TYPES = ['Transformation', 'Function'];

export function randomScript(ctx: any, apiVersion: string, runtime: any) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const type = pick(SCRIPT_TYPES);
  const language = runtime.language;
  const environment = randomAttribute('env');
  const code = randomAttribute('code');
  const params = null;
  const runtimeId = runtime.id;
  const projectId = runtime.projectId;
  const builtin = false;
  return {
    id,
    tenantId,
    name,
    description,
    type,
    language,
    environment,
    code,
    params,
    runtimeId,
    projectId,
    builtin,
  };
}

export function randomScriptUpdate(
  ctx: any,
  apiVersion: string,
  project: any,
  entity
) {
  const updated = randomScript(ctx, apiVersion, project);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyScript(script: any, apiVersion: string) {
  const {
    id,
    tenantId,
    name,
    description,
    type,
    language,
    environment,
    code,
    params,
    runtimeId,
    projectId,
    builtin,
  } = script;
  const doc: any = {
    id,
    tenantId,
    name,
    description,
    type,
    language,
    environment,
    code,
    params,
    runtimeId,
    projectId,
    builtin,
  };
  if (!params) {
    doc.params = null;
  }
  return doc;
}
