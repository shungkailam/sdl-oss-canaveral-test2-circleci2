import { range, randomCount, pick } from './common';
import { randomTenant } from './entities/tenant';
import { randomCategory } from './entities/category';
import { randomEdge } from './entities/edge';
import { randomDataSource } from './entities/dataSource';
import { randomCloudCreds } from './entities/cloudCreds';
import { randomDockerProfile } from './entities/dockerProfile';
import { randomProjectUser, randomAdminUser } from './entities/user';
import { randomProject } from './entities/project';
import { randomApplication } from './entities/application';
import { randomDataStream } from './entities/dataStream';
import { randomScript } from './entities/script';
import { randomScriptRuntime } from './entities/scriptRuntime';

// generate a context for a tenant
// context contains randomly generated entities of all types
// and serves as the playground for RBAC testing
export async function generate(tenantId: string, apiVersion: string) {
  const ctx: any = { name: 'global' };
  const tenant = await randomTenant(ctx, apiVersion, tenantId);

  ctx.tenant = tenant;
  ctx.tenantId = tenant.id;
  ctx.superUser = randomAdminUser(ctx, apiVersion);

  ctx.categories = range(randomCount(2, 6)).map(x =>
    randomCategory(ctx, apiVersion)
  );

  ctx.edges = range(randomCount(1, 2)).map(x => randomEdge(ctx, apiVersion));
  const dataSources: any[] = [];
  ctx.edges.forEach(edge => {
    range(randomCount(1, 2)).reduce((acc, cur) => {
      acc.push(randomDataSource(ctx, apiVersion, edge));
      return acc;
    }, dataSources);
  });
  ctx.dataSources = dataSources;
  ctx.cloudCredss = range(randomCount(2, 4)).map(x =>
    randomCloudCreds(ctx, apiVersion)
  );
  ctx.dockerProfiles = range(randomCount(3, 5)).map(x =>
    randomDockerProfile(ctx, apiVersion)
  );
  const users = [
    ...range(1).map(x => randomProjectUser(ctx, apiVersion)),
    ...range(1).map(x => randomAdminUser(ctx, apiVersion)),
  ];
  ctx.users = [ctx.superUser, ...users];

  ctx.projects = range(randomCount(1, 3)).map(x =>
    randomProject(ctx, apiVersion)
  );

  ctx.scriptRuntimes = [];
  ctx.scripts = [];
  ctx.applications = [];
  ctx.dataStreams = [];

  ctx.projects.forEach(project => {
    ctx.scriptRuntimes = ctx.scriptRuntimes.concat(
      range(randomCount(1, 2)).map(x =>
        randomScriptRuntime(ctx, apiVersion, project)
      )
    );

    ctx.applications = ctx.applications.concat(
      range(randomCount(1, 2)).map(x =>
        randomApplication(ctx, apiVersion, project)
      )
    );
  });

  ctx.scriptRuntimes.forEach(runtime => {
    ctx.scripts = ctx.scripts.concat(
      range(randomCount(1, 2)).map(x => randomScript(ctx, apiVersion, runtime))
    );
  });

  ctx.projects.forEach(project => {
    ctx.dataStreams = ctx.dataStreams.concat(
      range(randomCount(1, 2)).map(x =>
        randomDataStream(ctx, apiVersion, project)
      )
    );
  });

  // now fill in data stream origin id
  ctx.dataStreams.forEach(ds => {
    if (ds.origin === 'Data Stream') {
      ds.originId = pick(ctx.dataStreams).id;
    }
  });

  return ctx;
}

// user context captures what user should get when making REST GET calls
export function getUserContext(ctx, user) {
  const name = `user[${user.email}]`;
  const { tenant, tenantId, categories } = ctx;
  const isInfraAdmin = user.role === 'INFRA_ADMIN';
  const projects = ctx.projects.filter(p =>
    p.users.some(u => u.userId === user.id)
  );
  const allProjects = isInfraAdmin ? ctx.projects : projects;
  const edges = isInfraAdmin
    ? ctx.edges
    : ctx.edges.filter(e => projects.some(p => p.edgeIds.indexOf(e.id) !== -1));
  const dataSources = isInfraAdmin
    ? ctx.dataSources
    : ctx.dataSources.filter(d => edges.some(e => e.id === d.edgeId));
  const cloudCredss = isInfraAdmin
    ? ctx.cloudCredss
    : ctx.cloudCredss.filter(c =>
        projects.some(p => p.cloudCredentialIds.indexOf(c.id) !== -1)
      );
  const dockerProfiles = isInfraAdmin
    ? ctx.dockerProfiles
    : ctx.dockerProfiles.filter(d =>
        projects.some(p => p.dockerProfileIds.indexOf(d.id) !== -1)
      );
  // user can always get self even if not in any projects
  const users = isInfraAdmin
    ? ctx.users
    : ctx.users.filter(
        u =>
          u.id === user.id ||
          projects.some(p => p.users.some(pu => pu.userId === u.id))
      );
  if (!users.find(u => u.id === user.id)) {
    users.push(user);
  }
  const scriptRuntimes = ctx.scriptRuntimes.filter(sr =>
    projects.some(p => p.id === sr.projectId)
  );
  const scripts = ctx.scripts.filter(s =>
    projects.some(p => p.id === s.projectId)
  );
  const applications = ctx.applications.filter(a =>
    projects.some(p => p.id === a.projectId)
  );
  const dataStreams = ctx.dataStreams.filter(d =>
    projects.some(p => p.id === d.projectId)
  );

  return {
    name,
    user,
    tenant,
    tenantId,
    categories,
    edges,
    dataSources,
    cloudCredss,
    dockerProfiles,
    users,
    scriptRuntimes,
    scripts,
    applications,
    dataStreams,
    projects: allProjects,
    userProjects: projects,
  };
}

export function getEdgeContext(ctx, edge) {
  const name = `edge[${edge.id}]`;
  const { tenant, tenantId, categories } = ctx;
  const allProjects = ctx.projects;
  const projects = ctx.projects.filter(p => p.edgeIds.some(e => e === edge.id));
  // TODO FIXME - should we just let edge see itself, not all project members?
  // edge can always get self even if not in any projects
  const edges = ctx.edges.filter(
    e => e.id === edge.id || projects.some(p => p.edgeIds.indexOf(e.id) !== -1)
  );
  const dataSources = ctx.dataSources.filter(d =>
    edges.some(e => e.id === d.edgeId)
  );
  const cloudCredss = ctx.cloudCredss.filter(c =>
    projects.some(p => p.cloudCredentialIds.indexOf(c.id) !== -1)
  );
  const dockerProfiles = ctx.dockerProfiles.filter(d =>
    projects.some(p => p.dockerProfileIds.indexOf(d.id) !== -1)
  );
  const users = ctx.users.filter(u =>
    projects.some(p => p.users.some(pu => pu.userId === u.id))
  );
  const scriptRuntimes = ctx.scriptRuntimes.filter(sr =>
    projects.some(p => p.id === sr.projectId)
  );
  const scripts = ctx.scripts.filter(s =>
    projects.some(p => p.id === s.projectId)
  );
  const applications = ctx.applications.filter(a =>
    projects.some(p => p.id === a.projectId)
  );
  const dataStreams = ctx.dataStreams.filter(d =>
    projects.some(p => p.id === d.projectId)
  );
  return {
    name,
    edge,
    tenant,
    tenantId,
    categories,
    edges,
    dataSources,
    cloudCredss,
    dockerProfiles,
    users,
    scriptRuntimes,
    scripts,
    applications,
    dataStreams,
    projects: allProjects,
    userProjects: projects,
  };
}
