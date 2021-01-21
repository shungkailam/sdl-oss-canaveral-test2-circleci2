import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pick } from '../common';

export function randomDockerProfile(ctx: any, apiVersion: string) {
  const { tenantId, cloudCredss } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const server = randomAttribute('server') + '.b.c.d.e.f';
  const userName = randomAttribute('userName');
  const email = randomAttribute('email');
  const pwd = randomAttribute('pwd');
  const credentials = randomAttribute('credentials');
  const isCR = Math.random() < 0.3;
  const cloudCreds = pick(cloudCredss);
  const type = isCR ? 'ContainerRegistry' : cloudCreds.type;
  const cloudCredsID = isCR ? null : cloudCreds.id;
  if (apiVersion === 'v1') {
    return {
      id,
      tenantId,
      name,
      description,
      server,
      userName,
      email,
      pwd,
      credentials,
      type,
      cloudCredsID,
    };
  } else {
    const doc = {
      id,
      tenantId,
      name,
      description,
      server,
      credentials,
      type,
    };
    if (isCR) {
      return {
        ...doc,
        ContainerRegistryInfo: {
          email,
          pwd,
          userName,
        },
      };
    } else {
      return {
        ...doc,
        CloudProfileInfo: {
          cloudCredsID,
          email,
        },
      };
    }
  }
}

export function randomDockerProfileUpdate(
  ctx: any,
  apiVersion: string,
  entity
) {
  const updated = randomDockerProfile(ctx, apiVersion);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyDockerProfile(dockerProfile: any, apiVersion: string) {
  // note: ignore pwd, credentials, userName, type - since cloudmgmt mangles it
  // also drop email for now, since we use user email if have cloud creds id
  if (apiVersion === 'v1') {
    const {
      id,
      tenantId,
      name,
      description,
      server,
      // email,
      cloudCredsID,
    } = dockerProfile;
    return {
      id,
      tenantId,
      name,
      description,
      server,
      // email,
      cloudCredsID,
    };
  } else {
    const {
      id,
      tenantId,
      name,
      description,
      server,
      // email,
      ContainerRegistryInfo,
      CloudProfileInfo,
    } = dockerProfile;
    if (CloudProfileInfo) {
      const { cloudCredsID } = CloudProfileInfo;
      return {
        id,
        tenantId,
        name,
        description,
        server,
        CloudProfileInfo: { cloudCredsID },
      };
    } else {
      // drop pwd
      const { userName, email } = ContainerRegistryInfo;
      return {
        id,
        tenantId,
        name,
        description,
        server,
        ContainerRegistryInfo: { userName, email },
      };
    }
  }
}
