import { Sequelize } from 'sequelize-typescript';
import { QueryTypes } from 'sequelize';
import { initSequelize } from '../rest/sql-api/baseApi';

async function cleanupTenant(
  sql: Sequelize,
  tenantId: string,
  nHours: number
): Promise<void> {
  const dateStr = `now() - interval '${nHours} hour'`;
  console.log('date string:', dateStr);

  function getQuery(table: string): string {
    switch (table) {
      case 'script_model':
      case 'script_runtime_model':
        return `DELETE FROM ${table} WHERE tenant_id = '${tenantId}' and updated_at < ${dateStr} and builtin != True`;
      case 'category_model':
        return `DELETE FROM ${table} WHERE tenant_id = '${tenantId}' and updated_at < ${dateStr} and name != 'Data Type'`;
      case 'project_model':
        return `DELETE FROM ${table} WHERE tenant_id = '${tenantId}' and updated_at < ${dateStr} and name != 'Default Project' and name != 'compass' and name != 'Upgrade'`;
      case 'application_model':
        return `DELETE FROM ${table} WHERE tenant_id = '${tenantId}' and updated_at < ${dateStr} and name != 'deepomatic'`;
      case 'docker_profile_model':
        return `DELETE FROM ${table} WHERE tenant_id = '${tenantId}' and updated_at < ${dateStr} and name != 'deepomatic'`;
      default:
        return `DELETE FROM ${table} WHERE tenant_id = '${tenantId}' and updated_at < ${dateStr}`;
    }
  }

  function runCleanupQuery(table) {
    console.log(`cleanup from ${table} for tenant_id ${tenantId}...`);
    const q = getQuery(table);
    return sql.query(q, {
      type: QueryTypes.DELETE,
    });
  }

  try {
    await runCleanupQuery('script_model');
    await runCleanupQuery('script_runtime_model');
    await runCleanupQuery('sensor_model');
    await runCleanupQuery('application_status_model');
    await runCleanupQuery('application_model');
    await runCleanupQuery('data_stream_model');
    await runCleanupQuery('data_source_model');
    await runCleanupQuery('edge_cert_model');
    await runCleanupQuery('edge_model');
    await runCleanupQuery('edge_device_model');
    await runCleanupQuery('edge_cluster_model');
    await runCleanupQuery('edge_log_collect_model');
    await runCleanupQuery('docker_profile_model');
    await runCleanupQuery('cloud_creds_model');
    await runCleanupQuery('category_model');
    await runCleanupQuery('project_model');
    await runCleanupQuery('software_update_batch_model');
  } catch (e) {
    console.error('failed to clean up tenant:', e);
  }
}

//
// Script to cleanup a tenant by deleting objects in the tenant older than <n hours>.
//
const USAGE = `\nUsage: node cleanupTenant.js <tenant id> <n hours>\n`;

async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantId = process.argv[2];
  const nHours = parseInt(process.argv[3], 10);

  await cleanupTenant(sql, tenantId, nHours);

  sql.close();
}

main();
