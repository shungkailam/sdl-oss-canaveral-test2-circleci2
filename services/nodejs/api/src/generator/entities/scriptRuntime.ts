import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pick } from '../common';

const LANGUAGES = ['node', 'python', 'golang'];

export function randomScriptRuntime(
  ctx: any,
  apiVersion: string,
  project: any
) {
  const { tenantId, dockerProfiles } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const language = pick(LANGUAGES);
  const builtin = false;
  const dockerRepoURI = randomAttribute('dockerRepoURI');
  const projectId = project.id;
  const dockerProfileID = pick(
    dockerProfiles.filter(d => project.dockerProfileIds.indexOf(d.id) !== -1)
  ).id;
  const dockerfile = randomAttribute('dockerfile');
  return {
    id,
    tenantId,
    name,
    description,
    language,
    builtin,
    dockerRepoURI,
    dockerProfileID,
    dockerfile,
    projectId,
  };
}

export function randomScriptRuntimeUpdate(
  ctx: any,
  apiVersion: string,
  project: any,
  entity
) {
  const updated = randomScriptRuntime(ctx, apiVersion, project);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyScriptRuntime(scriptRuntime: any, apiVersion: string) {
  const {
    id,
    tenantId,
    name,
    description,
    language,
    builtin,
    dockerRepoURI,
    dockerProfileID,
    dockerfile,
    projectId,
  } = scriptRuntime;
  return {
    id,
    tenantId,
    name,
    description,
    language,
    builtin,
    dockerRepoURI,
    dockerProfileID,
    dockerfile,
    projectId,
  };
}
