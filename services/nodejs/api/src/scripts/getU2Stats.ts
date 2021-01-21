import * as AWS from 'aws-sdk';
import * as jwt from 'jsonwebtoken';
import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';
import { AWS_REGION } from '../rest/services/impl/key.service.aws';

// Note: on memory usage:
// The amount of data per tenant is small for this.
// To get data for 3000 tenants require about 1MB memory.
// When we are at > 100K tenants, we can optimize this to use stream approach.

//
// Script to gather hourly U2 stats and write output to S3.
//
const USAGE = `\nUsage: node getU2Stats.js go [<s3 bucket name>]\n`;

const DEFAULT_S3_BUCKET_NAME = 'u2-stats';

function getDate() {
  return new Date();
}

interface YMDH {
  year: string;
  month: string;
  day: string;
  hour: string;
}
// We don't use sprintf-js for now to minimize dependency
function getYMDH(d: Date): YMDH {
  const year = '' + d.getFullYear();
  let month = '' + (d.getMonth() + 1);
  let day = '' + d.getDate();
  let hour = '' + d.getHours();
  if (month.length === 1) {
    month = '0' + month;
  }
  if (day.length === 1) {
    day = '0' + day;
  }
  if (hour.length === 1) {
    hour = '0' + hour;
  }
  return { year, month, day, hour };
}
function getYMDHPath(d: Date): string {
  const ymdh = getYMDH(d);
  return `year=${ymdh.year}/month=${ymdh.month}/day=${ymdh.day}/hour=${
    ymdh.hour
  }`;
}
interface MM {
  minute: string;
}
function getMM(d: Date): MM {
  let minute = '' + d.getMinutes();
  if (minute.length === 1) {
    minute = '0' + minute;
  }
  return { minute };
}
function getMMSuffix(d: Date): string {
  const hm = getMM(d);
  return `_${hm.minute}`;
}

// poorman's version of Object.values
function objectValues(obj: { [key: string]: any }): any[] {
  return Object.keys(obj).map(tid => obj[tid]);
}

function serializeObjectArray(records: any[]): string {
  return records.map(r => JSON.stringify(r)).join('\n');
}

// One record per line for later Athena processing
function serializeMapEntries(
  entityMap: { [key: string]: any },
  sortFn: (a: any, b: any) => number
): string {
  const values = objectValues(entityMap);
  if (sortFn) {
    values.sort(sortFn);
  }
  return serializeObjectArray(values);
}

function getAuditLogTableName(d: Date): string {
  const ymd = getYMDH(d);
  return `audit_log_${ymd.year}${ymd.month}${ymd.day}`;
}

// query for select entity count group by tenant id
function qs(model: string): string {
  return `SELECT tenant_id, count(*) from ${model} group by tenant_id`;
}

function getS3(): AWS.S3 {
  AWS.config.update(<any>{
    region: AWS_REGION,
  });
  return new AWS.S3({ apiVersion: '2006-03-01' });
}
function getBucketKey(d: Date, prefix: string): string {
  return `${prefix}/${getYMDHPath(d)}/${prefix}${getMMSuffix(d)}.txt`;
}
// check if bucket exists and can access
async function headBucket(s3: AWS.S3, name: string): Promise<boolean> {
  try {
    await s3.headBucket({ Bucket: name }).promise();
    return true;
  } catch (e) {
    return false;
  }
}
// create bucket if it does not exist
async function ensureBucket(s3: AWS.S3, name: string, region: string) {
  const ok = await headBucket(s3, name);
  if (!ok) {
    // create bucket
    const params = {
      Bucket: name,
      CreateBucketConfiguration: {
        LocationConstraint: region,
      },
    };
    return await s3.createBucket(params).promise();
  }
  return null;
}

async function putS3Object(
  s3: AWS.S3,
  bucketName: string,
  prefix: string,
  d: Date,
  objectBody: string
) {
  const key = getBucketKey(d, prefix);
  return await s3
    .putObject({
      Body: objectBody,
      Bucket: bucketName,
      Key: key,
      Tagging: '',
    })
    .promise();
}

interface EntityCount {
  tenantId: string;
  tenantName: string;
  tenantDescription: string;
  myNutanixId: string;
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
  totalCount: number;
}

interface TenantData {
  id: string;
  name: string;
  description: string;
  external_id: string;
}

function findTenant(tenants: TenantData[], tenantId: string): TenantData {
  const tenant = tenants.find(t => t.id === tenantId);
  if (tenant) {
    return tenant;
  }
  // tenant has since been deleted
  return {
    id: tenantId,
    name: 'N/A',
    description: 'N/A',
    external_id: null,
  };
}

async function fetchTenants(sql: any): Promise<TenantData[]> {
  return await doQuery(
    sql,
    `SELECT id, name, description, external_id from tenant_model`
  );
}

async function fetchEntityCounts(
  sql: any,
  tenants: TenantData[]
): Promise<{ [key: string]: EntityCount }> {
  const entityCounts: { [key: string]: EntityCount } = {};
  tenants.forEach(e => {
    entityCounts[e.id] = {
      tenantId: e.id,
      tenantName: e.name,
      tenantDescription: e.description,
      myNutanixId: e.external_id,
      edgeCount: 0,
      userCount: 0,
      categoryCount: 0,
      cloudCredsCount: 0,
      dockerProfileCount: 0,
      dataSourceCount: 0,
      dataStreamCount: 0,
      projectCount: 0,
      scriptCount: 0,
      scriptRuntimeCount: 0,
      appCount: 0,
      mlModelCount: 0,
      sensorCount: 0,
      totalCount: 0,
    };
  });

  // entity count to model map
  const ecm = {
    edgeCount: 'edge_model',
    userCount: 'user_model',
    appCount: 'application_model',
    categoryCount: 'category_model',
    cloudCredsCount: 'cloud_creds_model',
    dockerProfileCount: 'docker_profile_model',
    dataSourceCount: 'data_source_model',
    dataStreamCount: 'data_stream_model',
    projectCount: 'project_model',
    scriptCount: 'script_model',
    scriptRuntimeCount: 'script_runtime_model',
    mlModelCount: 'machine_inference_model',
    sensorCount: 'sensor_model',
  };
  const keys = Object.keys(ecm);
  // fetch count for all entity types
  const rawEntitiesCounts = await Promise.all(
    keys.map(key => doQuery(sql, qs(ecm[key])))
  );
  // convert string count to number
  rawEntitiesCounts.forEach((rawEntityCounts, index) => {
    const key = keys[index];
    rawEntityCounts.forEach(
      e => (entityCounts[e.tenant_id][key] = Number(e.count))
    );
  });
  // add count for each entity type to total count
  Object.keys(entityCounts).forEach(tid => {
    const ec = entityCounts[tid];
    keys.forEach(k => (ec.totalCount += ec[k]));
  });
  return entityCounts;
}

async function fetchQRCodeStats(sql: any, d: Date, tenants: TenantData[]) {
  const tableName = getAuditLogTableName(d);
  const records: any = {};
  // short token does not contain tenant info, so fetch it from audit log row directly
  // also since expiry will be the same as corresponding qr code,
  // so for simplicity don't replicate it here to avoid parsing the token in response_message
  const q = `select tenant_id, user_email, count(request_id) from ${tableName} where request_method = 'POST' and request_url = '/v1.0/login/shortlogintoken' and response_code = 200 group by tenant_id, user_email`;
  (await sql.query(q, {
    type: sql.QueryTypes.SELECT,
  })).forEach(i => {
    try {
      const { tenant_id: tenantId, user_email: email, count: sCount } = i;
      const count = Number(sCount);
      const {
        name: tenantName,
        description: tenantDescription,
        external_id: myNutanixId,
      } = findTenant(tenants, tenantId);

      let record = records[tenantId];
      if (!record) {
        record = records[tenantId] = {
          tenantId,
          tenantName,
          tenantDescription,
          myNutanixId,
          totalCount: 0,
          breakdown: [],
        };
      }
      record.totalCount += count;
      record.breakdown.push({
        userEmail: email,
        count,
      });
    } catch (e) {
      // ignore
    }
  });
  return records;
}

async function fetchStatsCommon(
  sql: any,
  d: Date,
  url: string,
  tenants: TenantData[]
) {
  const tableName = getAuditLogTableName(d);
  const records: any = {};
  const q = `select response_message from ${tableName} where request_method = 'POST' and request_url = '${url}' and response_code = 200`;
  (await sql.query(q, {
    type: sql.QueryTypes.SELECT,
  })).forEach(i => {
    try {
      const { _id: id, name, email, token } = JSON.parse(i.response_message);
      const { tenantId, trialExpiry } = <any>jwt.decode(token);
      const {
        name: tenantName,
        description: tenantDescription,
        external_id: myNutanixId,
      } = findTenant(tenants, tenantId);
      let record = records[tenantId];
      if (!record) {
        record = records[tenantId] = {
          tenantId,
          tenantName,
          tenantDescription,
          myNutanixId,
          totalCount: 0,
          breakdown: {},
        };
        // trialExpiry is optional, only add it if present
        if (trialExpiry) {
          record['trialExpiry'] = records[tenantId][
            'trialExpiry'
          ] = trialExpiry;
        }
      }
      record.totalCount++;
      let userStats = record.breakdown[id];
      if (!userStats) {
        userStats = record.breakdown[id] = {
          userId: id,
          userName: name,
          userEmail: email,
          count: 0,
        };
      }
      userStats.count++;
    } catch (e) {
      // ignore
    }
  });
  // convert breakdown from object to array
  Object.keys(records).forEach(tid => {
    const record = records[tid];
    if (record.breakdown) {
      record.breakdown = objectValues(record.breakdown);
    }
  });
  return records;
}
function fetchLoginStats(sql: any, d: Date, tenants: TenantData[]) {
  return fetchStatsCommon(sql, d, '/v1/oauth2/token', tenants);
}
function fetchMobileLoginStats(sql: any, d: Date, tenants: TenantData[]) {
  return fetchStatsCommon(sql, d, '/v1.0/login/logintoken', tenants);
}

async function publishEntityCounts(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date,
  tenants: TenantData[]
) {
  const entityCounts = await fetchEntityCounts(sql, tenants);
  const objectBody = serializeMapEntries(
    entityCounts,
    (a: EntityCount, b: EntityCount) => b.totalCount - a.totalCount
  );
  await putS3Object(s3, bucketName, 'entity-count', d, objectBody);
}

async function publishStatsCommon(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  keyPrefix: string,
  d: Date,
  tenants: TenantData[],
  fn: (sql: any, d: Date, tenants: TenantData[]) => any
) {
  const records = await fn(sql, d, tenants);
  const objectBody = serializeMapEntries(
    records,
    (a: any, b: any) => b.count - a.count
  );
  await putS3Object(s3, bucketName, keyPrefix, d, objectBody);
}
function publishLoginStats(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date,
  tenants: TenantData[]
) {
  return publishStatsCommon(
    sql,
    s3,
    bucketName,
    'login-stats',
    d,
    tenants,
    fetchLoginStats
  );
}
function publishMobileLoginStats(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date,
  tenants: TenantData[]
) {
  return publishStatsCommon(
    sql,
    s3,
    bucketName,
    'mobile-login-stats',
    d,
    tenants,
    fetchMobileLoginStats
  );
}
function publishQRCodeStats(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date,
  tenants: TenantData[]
) {
  return publishStatsCommon(
    sql,
    s3,
    bucketName,
    'qr-code-stats',
    d,
    tenants,
    fetchQRCodeStats
  );
}

async function publishTimedModelCommon(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date,
  modelName: string,
  columnName: string
) {
  const ymdh = getYMDH(d);
  const ds = `${ymdh.year}-${ymdh.month}-${ymdh.day}`;
  const hh = d.getHours();
  const start = `${ds} ${hh - 1}:00`;
  const end = `${ds} ${hh}:00`;
  const records = await doQuery(
    sql,
    `SELECT * from ${modelName} WHERE ${columnName} > '${start}' and ${columnName} < '${end}'`
  );
  const objectBody = serializeObjectArray(records);
  return await putS3Object(s3, bucketName, modelName, d, objectBody);
}
async function publishModelCommon(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date,
  modelName: string
) {
  const records = await doQuery(sql, `SELECT * from ${modelName}`);
  const objectBody = serializeObjectArray(records);
  return await putS3Object(s3, bucketName, modelName, d, objectBody);
}
function publishTpsRegistrationModel(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date
) {
  return publishModelCommon(sql, s3, bucketName, d, 'tps_registration_model');
}
function publishTpsTenantPoolModel(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date
) {
  return publishModelCommon(sql, s3, bucketName, d, 'tps_tenant_pool_model');
}
function publishTpsEdgeContextModel(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date
) {
  return publishModelCommon(sql, s3, bucketName, d, 'tps_edge_context_model');
}
function publishTpsAuditLogModel(
  sql: any,
  s3: AWS.S3,
  bucketName: string,
  d: Date
) {
  return publishTimedModelCommon(
    sql,
    s3,
    bucketName,
    d,
    'tps_audit_log_model',
    'created_at'
  );
}

function syncAthenaTables(): Promise<any> {
  const athena = new AWS.Athena();
  const params: AWS.Athena.StartQueryExecutionInput[] = [
    {
      QueryString: 'MSCK REPAIR TABLE u2_entity_counts_prod',
      ResultConfiguration: {
        OutputLocation: 's3://sherlock-u2-stats-prod2dev/output/entity-count/',
      },
    },
    {
      QueryString: 'MSCK REPAIR TABLE u2_login_stats_prod',
      ResultConfiguration: {
        OutputLocation: 's3://sherlock-u2-stats-prod2dev/output/login-stats/',
      },
    },
    {
      QueryString: 'MSCK REPAIR TABLE u2_mobile_login_stats_prod',
      ResultConfiguration: {
        OutputLocation:
          's3://sherlock-u2-stats-prod2dev/output/mobile-login-stats/',
      },
    },
    {
      QueryString: 'MSCK REPAIR TABLE u2_qr_code_stats_prod',
      ResultConfiguration: {
        OutputLocation: 's3://sherlock-u2-stats-prod2dev/output/qr-code-stats/',
      },
    },
    {
      QueryString: 'MSCK REPAIR TABLE tps_tenant_pool_model_prod',
      ResultConfiguration: {
        OutputLocation:
          's3://sherlock-u2-stats-prod2dev/output/tps_tenant_pool_model/',
      },
    },
    {
      QueryString: 'MSCK REPAIR TABLE tps_edge_context_model_prod',
      ResultConfiguration: {
        OutputLocation:
          's3://sherlock-u2-stats-prod2dev/output/tps_edge_context_model/',
      },
    },
    {
      QueryString: 'MSCK REPAIR TABLE tps_registration_model_prod',
      ResultConfiguration: {
        OutputLocation:
          's3://sherlock-u2-stats-prod2dev/output/tps_registration_model/',
      },
    },
  ];
  return Promise.all(
    params.map(param => athena.startQueryExecution(param).promise())
  );
}

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }
  let sql = initSequelize();
  let failed = false;
  try {
    const s3 = getS3();
    const bucketName = process.argv[3] || DEFAULT_S3_BUCKET_NAME;
    const d = getDate();
    await ensureBucket(s3, bucketName, AWS_REGION);
    const tenants = await fetchTenants(sql);
    await publishEntityCounts(sql, s3, bucketName, d, tenants);
    await publishLoginStats(sql, s3, bucketName, d, tenants);
    await publishMobileLoginStats(sql, s3, bucketName, d, tenants);
    await publishQRCodeStats(sql, s3, bucketName, d, tenants);
    await publishTpsRegistrationModel(sql, s3, bucketName, d);
    await publishTpsTenantPoolModel(sql, s3, bucketName, d);
    await publishTpsEdgeContextModel(sql, s3, bucketName, d);
    await publishTpsAuditLogModel(sql, s3, bucketName, d);
    await syncAthenaTables();
  } catch (e) {
    console.log('Caught exception:', e);
    failed = true;
  }
  sql.close();
  if (failed) {
    process.exit(1);
  }
}

main();
