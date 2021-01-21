import * as md5 from 'md5';

export function getDefaultProjectId(tenantId: string): string {
  return md5(`${tenantId}/default-project`);
}

export function doQuery(sql: any, query) {
  return sql.query(query, { type: sql.QueryTypes.SELECT });
}

export function doUpdate(sql: any, query) {
  return sql.query(query, { type: sql.QueryTypes.UPDATE });
}

export async function assignInfraAdminRoleToAllUsers(
  sql: any,
  tenantId: string
) {
  return doUpdate(
    sql,
    `UPDATE user_model SET role = 'INFRA_ADMIN' WHERE tenant_id = '${tenantId}'`
  );
}

export async function getAllTenantIDs(sql: any): Promise<string[]> {
  const tenantIDs = (await doQuery(sql, `SELECT id from tenant_model`)).map(
    e => e.id
  );
  return tenantIDs;
}

export async function getTenant(sql: any, tenantId: string): Promise<any> {
  const rs = await doQuery(
    sql,
    `SELECT * from tenant_model WHERE id='${tenantId}'`
  );
  if (rs && rs.length) {
    return rs[0];
  }
  return null;
}

export async function createDefaultProject(
  sql: any,
  tenantId: string,
  forceUpdate: boolean
) {
  const projectId = getDefaultProjectId(tenantId);

  const projects = await doQuery(
    sql,
    `SELECT * FROM project_model WHERE tenant_id = '${tenantId}'`
  );

  // only create default project if no projects exist
  if (projects.length === 0) {
    const edgeIds = (await doQuery(
      sql,
      `SELECT id from edge_model WHERE tenant_id = '${tenantId}'`
    )).map(e => e.id);
    console.log(`tenant id ${tenantId}, edgeIds: ${edgeIds}`);
    await sql.query(
      `INSERT INTO project_model (id, version, tenant_id, name, description, edge_selector_type, edge_selectors, created_at, updated_at) VALUES ('${projectId}', 1, '${tenantId}', 'Default Project', 'Default Project for backward compatibility', 'Explicit', '[]', current_timestamp, current_timestamp)`,
      { type: sql.QueryTypes.INSERT }
    );
    console.log(`done create default project for tenant id ${tenantId}`);

    // also assign INFRA_ADMIN to all users
    await assignInfraAdminRoleToAllUsers(sql, tenantId);

    // add all users to default project
    const userIds = (await doQuery(
      sql,
      `SELECT id from user_model WHERE tenant_id = '${tenantId}'`
    )).map(e => e.id);
    await Promise.all(
      userIds.map(id =>
        sql.query(
          `INSERT INTO project_user_model (project_id, user_id, user_role) VALUES ('${projectId}', '${id}', 'PROJECT_ADMIN')`,
          { type: sql.QueryTypes.INSERT }
        )
      )
    );

    // add all cloud profiles to default project
    const cloudProfileIds = (await doQuery(
      sql,
      `SELECT id from cloud_creds_model WHERE tenant_id = '${tenantId}'`
    )).map(e => e.id);
    await Promise.all(
      cloudProfileIds.map(id =>
        sql.query(
          `INSERT INTO project_cloud_creds_model (project_id, cloud_creds_id) VALUES ('${projectId}', '${id}')`,
          { type: sql.QueryTypes.INSERT }
        )
      )
    );

    // add all docker profiles to default project
    const dockerProfileIds = (await doQuery(
      sql,
      `SELECT id from docker_profile_model WHERE tenant_id = '${tenantId}'`
    )).map(e => e.id);
    await Promise.all(
      dockerProfileIds.map(id =>
        sql.query(
          `INSERT INTO project_docker_profile_model (project_id, docker_profile_id) VALUES ('${projectId}', '${id}')`,
          { type: sql.QueryTypes.INSERT }
        )
      )
    );

    // add all edges to default project
    await Promise.all(
      edgeIds.map(id =>
        sql.query(
          `INSERT INTO project_edge_model (project_id, edge_id) VALUES ('${projectId}', '${id}')`,
          { type: sql.QueryTypes.INSERT }
        )
      )
    );
  } else {
    console.log(
      `skip default project creation, project exists for tenant id ${tenantId}`
    );
  }

  if (projects.length === 0 || forceUpdate) {
    // see if some entity have no project id
    const noProjApps = await doQuery(
      sql,
      `SELECT ID FROM application_model WHERE tenant_id = '${tenantId}' AND project_id is NULL`
    );
    const noProjDataStreams = await doQuery(
      sql,
      `SELECT ID FROM data_stream_model WHERE tenant_id = '${tenantId}' AND project_id is NULL`
    );
    const noProjScripts = await doQuery(
      sql,
      `SELECT ID FROM script_model WHERE tenant_id = '${tenantId}' AND builtin != TRUE AND project_id is NULL`
    );
    const noProjScriptRuntimes = await doQuery(
      sql,
      `SELECT ID FROM script_runtime_model WHERE tenant_id = '${tenantId}' AND builtin != TRUE AND project_id is NULL`
    );

    if (
      noProjApps.length ||
      noProjDataStreams.length ||
      noProjScripts.length ||
      noProjScriptRuntimes.length
    ) {
      if (noProjApps.length) {
        await sql.query(
          `UPDATE application_model SET project_id = '${projectId}' WHERE tenant_id = '${tenantId}' AND project_id is NULL`,
          { type: sql.QueryTypes.UPDATE }
        );
      }
      if (noProjDataStreams.length) {
        await sql.query(
          `UPDATE data_stream_model SET project_id = '${projectId}' WHERE tenant_id = '${tenantId}' AND project_id is NULL`,
          { type: sql.QueryTypes.UPDATE }
        );
      }
      if (noProjScripts.length) {
        await sql.query(
          `UPDATE script_model SET project_id = '${projectId}' WHERE tenant_id = '${tenantId}' AND builtin != TRUE AND project_id is NULL`,
          { type: sql.QueryTypes.UPDATE }
        );
      }
      if (noProjScriptRuntimes.length) {
        await sql.query(
          `UPDATE script_runtime_model SET project_id = '${projectId}' WHERE tenant_id = '${tenantId}' AND builtin != TRUE AND project_id is NULL`,
          { type: sql.QueryTypes.UPDATE }
        );
      }
      console.log(
        `done update default project associations for tenant id ${tenantId}`
      );
    }
  }
}
