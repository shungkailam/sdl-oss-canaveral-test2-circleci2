import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';

//
// Script to get entity counts for all tenants
//
// Output is json of the form: (tenant id -> tenant entity counts)
// {
//    "19d0e1ec-f9b8-11e8-b7e3-506b8da26edc": {
//      "tenantId": "19d0e1ec-f9b8-11e8-b7e3-506b8da26edc",
//      "tenantName": "test-tenant-root-2018-12-06-16-36-13",
//      "edgeCount": 4,
//      "userCount": 1,
//      "categoryCount": 3,
//      "cloudCredsCount": 2,
//      "dockerProfileCount": 1,
//      "dataSourceCount": 1,
//      "dataStreamCount": 1,
//      "projectCount": 1,
//      "scriptCount": 3,
//      "scriptRuntimeCount": 5
//    }
// }
const USAGE = `\nUsage: node getTenantsEntityCounts.js go\n`;

function qs(model) {
  return `SELECT tenant_id, count(*) from ${model} group by tenant_id`;
}
interface EntityCount {
  tenantId: string;
  tenantName: string;
  edgeCount: number;
  userCount: number;
  categoryCount: number;
  cloudCredsCount: number;
  dockerProfileCount: number;
  dataSourceCount: number;
  dataStreamCount: number;
  projectCount: number;
  scriptCount: number;
  scriptRuntimeCount: number;
  appCount: number;
  mlModelCount: number;
  sensorCount: number;
}
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenants = await doQuery(sql, `SELECT id, name from tenant_model`);
  const edgeCounts = await doQuery(sql, qs('edge_cluster_model'));
  const userCounts = await doQuery(sql, qs('user_model'));
  const appCounts = await doQuery(sql, qs('application_model'));
  const categoryCounts = await doQuery(sql, qs('category_model'));
  const cloudCredsCounts = await doQuery(sql, qs('cloud_creds_model'));
  const dockerProfileCounts = await doQuery(sql, qs('docker_profile_model'));
  const dataSourceCounts = await doQuery(sql, qs('data_source_model'));
  const dataStreamCounts = await doQuery(sql, qs('data_stream_model'));
  const projectCounts = await doQuery(sql, qs('project_model'));
  const scriptCounts = await doQuery(sql, qs('script_model'));
  const scriptRuntimeCounts = await doQuery(sql, qs('script_runtime_model'));
  const mlModelCounts = await doQuery(sql, qs('machine_inference_model'));
  const sensorCounts = await doQuery(sql, qs('sensor_model'));

  const entityCounts: { [key: string]: EntityCount } = {};
  tenants.forEach(e => {
    entityCounts[e.id] = <EntityCount>{
      tenantId: e.id,
      tenantName: e.name,
    };
  });
  edgeCounts.forEach(
    e => (entityCounts[e.tenant_id].edgeCount = Number(e.count))
  );
  userCounts.forEach(
    e => (entityCounts[e.tenant_id].userCount = Number(e.count))
  );
  appCounts.forEach(
    e => (entityCounts[e.tenant_id].appCount = Number(e.count))
  );
  categoryCounts.forEach(
    e => (entityCounts[e.tenant_id].categoryCount = Number(e.count))
  );
  cloudCredsCounts.forEach(
    e => (entityCounts[e.tenant_id].cloudCredsCount = Number(e.count))
  );
  dockerProfileCounts.forEach(
    e => (entityCounts[e.tenant_id].dockerProfileCount = Number(e.count))
  );
  dataSourceCounts.forEach(
    e => (entityCounts[e.tenant_id].dataSourceCount = Number(e.count))
  );
  dataStreamCounts.forEach(
    e => (entityCounts[e.tenant_id].dataStreamCount = Number(e.count))
  );
  projectCounts.forEach(
    e => (entityCounts[e.tenant_id].projectCount = Number(e.count))
  );
  scriptCounts.forEach(
    e => (entityCounts[e.tenant_id].scriptCount = Number(e.count))
  );
  scriptRuntimeCounts.forEach(
    e => (entityCounts[e.tenant_id].scriptRuntimeCount = Number(e.count))
  );
  mlModelCounts.forEach(
    e => (entityCounts[e.tenant_id].mlModelCount = Number(e.count))
  );
  sensorCounts.forEach(
    e => (entityCounts[e.tenant_id].sensorCount = Number(e.count))
  );

  console.log(JSON.stringify(entityCounts, null, 2));

  sql.close();
}

main();
