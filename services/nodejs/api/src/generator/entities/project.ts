import * as uuidv4 from 'uuid/v4';
import { randomAttribute, pickMany } from '../common';

function makeProjectUserInfo(user) {
  const userId = user.id;
  const role = 'PROJECT_ADMIN';
  return {
    userId,
    role,
  };
}

function userSortFn(a, b) {
  return a.userId.localeCompare(b.userId);
}

export function randomProject(ctx: any, apiVersion: string) {
  const { tenantId, edges, cloudCredss, dockerProfiles, users: allUsers } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const description = randomAttribute('description');
  const cloudCredentialIds = pickMany(cloudCredss)
    .map(c => c.id)
    .sort();
  const dockerProfileIds = pickMany(dockerProfiles)
    .map(d => d.id)
    .sort();
  // fix for new change where adding a docker profile to project
  // will also add its backing cloud profile to project
  // So, for each docker profile backed by a cloud profile, add the cloud
  // profile to project if not already in
  dockerProfiles.forEach(dockerProfile => {
    if (dockerProfileIds.indexOf(dockerProfile.id) !== -1) {
      if (apiVersion === 'v1') {
        if (dockerProfile.cloudCredsID) {
          if (cloudCredentialIds.indexOf(dockerProfile.cloudCredsID) == -1) {
            cloudCredentialIds.push(dockerProfile.cloudCredsID);
            cloudCredentialIds.sort();
          }
        }
      } else {
        if (dockerProfile.CloudProfileInfo) {
          if (
            cloudCredentialIds.indexOf(
              dockerProfile.CloudProfileInfo.cloudCredsID
            ) == -1
          ) {
            cloudCredentialIds.push(
              dockerProfile.CloudProfileInfo.cloudCredsID
            );
            cloudCredentialIds.sort();
          }
        }
      }
    }
  });
  const [superUser, ...otherUsers] = allUsers;
  const users = [
    makeProjectUserInfo(superUser),
    ...pickMany(otherUsers).map(u => makeProjectUserInfo(u)),
  ].sort(userSortFn);
  const edgeSelectorType = 'Explicit'; // 'Explicit' | 'Category'
  const edgeIds = pickMany(edges)
    .map(e => e.id)
    .sort();
  const edgeSelectors = null;
  return {
    id,
    tenantId,
    name,
    description,
    cloudCredentialIds,
    dockerProfileIds,
    users,
    edgeSelectorType,
    edgeIds,
    edgeSelectors,
  };
}

export function randomProjectUpdate(ctx: any, apiVersion: string, entity) {
  const updated = randomProject(ctx, apiVersion);
  const { id } = entity;
  return { ...updated, id };
}

export function purifyProject(project: any, apiVersion: string) {
  const {
    id,
    tenantId,
    name,
    description,
    cloudCredentialIds,
    dockerProfileIds,
    users,
    edgeSelectorType,
    edgeIds,
    edgeSelectors,
  } = project;
  if (users) {
    users.sort(userSortFn);
  }
  if (edgeIds) {
    edgeIds.sort();
  }
  if (cloudCredentialIds) {
    cloudCredentialIds.sort();
  }
  if (dockerProfileIds) {
    dockerProfileIds.sort();
  }
  const doc: any = {
    id,
    tenantId,
    name,
    description,
    cloudCredentialIds,
    dockerProfileIds,
    users,
    edgeSelectorType,
    edgeIds,
    edgeSelectors,
  };
  if (!edgeSelectors) {
    doc.edgeSelectors = null;
  }
  return doc;
}
