import { Sequelize } from 'sequelize-typescript';
import { QueryTypes } from 'sequelize';

// disableTenant: disable a tenant by removing the external ID association,
// renaming its users and modifying existing edge serial numbers.
export async function disableTenant(
  sql: Sequelize,
  tenantId: string
): Promise<void> {
  console.log(`Disabling tenant ${tenantId}`);
  const now = new Date();
  const dbTimestamp = getDBTimestamp(now);
  const epochSeconds = getEpochSeconds(now);
  const updateTenant = `UPDATE tenant_model SET external_id=null, updated_at='${dbTimestamp}' WHERE id='${tenantId}'`;
  const renameEmail = `UPDATE user_model SET email=CONCAT(email, '.', ${epochSeconds}, '.ntnx-del'), updated_at='${dbTimestamp}' WHERE tenant_id='${tenantId}' AND email not like '%.ntnx-del'`;
  const updateEdgeSerial = `UPDATE edge_model SET serial_number=CONCAT(serial_number, '.', ${epochSeconds}, '.ntnx-del'), updated_at='${dbTimestamp}' WHERE tenant_id='${tenantId}' AND serial_number not like '%.ntnx-del'`;
  const existEdgeDeviceTable = `SELECT EXISTS ( SELECT 1 FROM information_schema.tables WHERE  table_schema = 'public' AND table_name = 'edge_device_model');`;
  const updateEdgeDeviceSerial = `UPDATE edge_device_model SET serial_number=CONCAT(serial_number, '.', ${epochSeconds}, '.ntnx-del'), updated_at='${dbTimestamp}' WHERE tenant_id='${tenantId}' AND serial_number not like '%.ntnx-del'`;
  // Trigger deletion of trial edge if any
  const updateTenantPoolModel = `UPDATE tps_tenant_pool_model SET state = 'DELETING', updated_at='${dbTimestamp}' WHERE id='${tenantId}' AND state != 'DELETING'`;

  await sql
    .transaction(async transaction => {
      console.log('Updating tenant model');
      await sql.query(updateTenant, {
        type: QueryTypes.UPDATE,
        transaction,
        raw: true,
      });
      console.log('Updating user model');
      await sql.query(renameEmail, {
        type: QueryTypes.UPDATE,
        transaction,
        raw: true,
      });
      console.log('Updating tenantpool model');
      await sql.query(updateTenantPoolModel, {
        type: QueryTypes.UPDATE,
        transaction,
        raw: true,
      });
      await sql
        .query(existEdgeDeviceTable, {
          type: QueryTypes.SELECT,
          transaction,
          raw: true,
        })
        .then(function(result) {
          if (result) {
            console.log('Updating edge device model');
            return new Promise((resolve, reject) => {
              try {
                sql.query(updateEdgeDeviceSerial, {
                  type: QueryTypes.UPDATE,
                  transaction,
                  raw: true,
                });
                resolve();
              } catch (e) {
                reject();
              }
            });
          }
          console.log('Updating edge model');
          return new Promise((resolve, reject) => {
            try {
              sql.query(updateEdgeSerial, {
                type: QueryTypes.UPDATE,
                transaction,
                raw: true,
              });
              resolve();
            } catch (e) {
              reject();
            }
          });
        });
    })
    .then(() => {
      console.log('Successfully disabled tenant');
    })
    .catch(err => {
      console.error('failed to disable tenant:', err);
    });
}

export async function getAuditLogTableNames(sql: Sequelize, tenantId: string) {
  const q = `SELECT table_name from audit_log_from_request_id WHERE tenant_id = '${tenantId}' group by table_name`;
  return (await sql.query(q, {
    type: QueryTypes.SELECT,
  })).map(i => (<any>i).table_name);
}

// deleteTenant: use raw DB delete for deleting a tenant
export async function deleteTenant(
  sql: Sequelize,
  tenantId: string
): Promise<void> {
  function runDeleteQuery(table) {
    console.log(`delete from ${table} for tenant_id ${tenantId}...`);
    const q =
      table === 'tenant_model'
        ? `DELETE FROM tenant_model WHERE id = '${tenantId}'`
        : `DELETE FROM ${table} WHERE tenant_id = '${tenantId}'`;
    return sql.query(q, {
      type: QueryTypes.DELETE,
    });
  }

  try {
    await runDeleteQuery('log_model');
    await runDeleteQuery('script_model');
    await runDeleteQuery('script_runtime_model');
    await runDeleteQuery('sensor_model');
    await runDeleteQuery('application_status_model');
    await runDeleteQuery('application_model');
    await runDeleteQuery('data_stream_model');
    await runDeleteQuery('data_source_model');
    await runDeleteQuery('edge_cert_model');
    await runDeleteQuery('edge_model');
    await runDeleteQuery('edge_device_model');
    await runDeleteQuery('edge_cluster_model');
    await runDeleteQuery('edge_log_collect_model');
    await runDeleteQuery('docker_profile_model');
    await runDeleteQuery('cloud_creds_model');
    await runDeleteQuery('category_model');
    await runDeleteQuery('project_model');
    await runDeleteQuery('user_model');
    await runDeleteQuery('software_update_batch_model');
    const auditLogTableNames = await getAuditLogTableNames(sql, tenantId);
    await Promise.all(
      auditLogTableNames.map(tableName => runDeleteQuery(tableName))
    );
    await runDeleteQuery('audit_log_from_request_id');
    await runDeleteQuery('storage_profile_model');
    await runDeleteQuery('tenant_model');
  } catch (e) {
    console.error('failed to delete tenant:', e);
  }
}

// promise will start executing once it is constructed.
// To avoid having too many promises executing at the same time,
// this function takes a promiseGenFn function to construct promise
// from each arg in the args array in a staggered fashion
export async function staggerPromises(
  args: any[],
  promiseGenFn: any,
  N,
  callback
) {
  let results = [];
  if (args.length <= N) {
    results = await Promise.all(args.map(a => promiseGenFn(a)));
    if (callback) {
      await callback();
    }
  } else {
    const n = Math.floor((args.length + N - 1) / N);
    for (let i = 0; i < n; i++) {
      const ps = args.slice(i * N, (i + 1) * N);
      const psa = await Promise.all(ps.map(a => promiseGenFn(a)));
      results = results.concat(psa);
      // invoke callback
      if (i < n - 1) {
        if (callback) {
          await callback();
        }
      }
    }
  }
  return results;
}

function randomFraction(max): number {
  return Math.round(Math.random() * 100 * max) / 100;
}
function randomPercent(max): number {
  return Math.round(Math.random() * 1000 * max) / 10;
}

export function genEdgeInfo(tmpl: any) {
  const totalMem = 16 * (1 << 20);
  const freeMem = (1 - randomFraction(0.4)) * totalMem;
  const totalStore = 2 * (1 << 30);
  const freeStore = (1 - randomFraction(0.3)) * totalStore;
  return {
    ...tmpl,
    NumCPU: '4',
    TotalMemoryKB: '' + totalMem,
    TotalStorageKB: '' + totalStore,
    GPUInfo: 'NVIDIA',
    CPUUsage: '' + randomPercent(0.3),
    MemoryFreeKB: '' + freeMem,
    StorageFreeKB: '' + freeStore,
    GPUUsage: '' + randomPercent(0.2),
  };
}

export function getEpochSeconds(date: Date): number {
  return Math.round(date.getTime() / 1000);
}

export function getDBTimestamp(date: Date): string {
  // '2012-11-04T14:51:06.157Z' to '2012-11-04 14:51:06.157'
  return date
    .toISOString()
    .replace(/T/, ' ')
    .replace(/Z/, '');
}
