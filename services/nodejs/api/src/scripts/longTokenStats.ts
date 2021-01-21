import { initSequelize } from '../rest/sql-api/baseApi';
import * as jwt from 'jsonwebtoken';

//
// Script to get (long)logintoken stats (launch mobile app by reading QR code) across all tenants
//
// Output format:
//   {
//      <tenant id>: {
//         count,
//         breakdown: {
//           <user_id>: {<name>, <email>, <count>},
//           ...
//         }
//      },
//      ...
//   }
//
const USAGE = `\nUsage: node longTokenStats.js <table name, e.g., audit_log_20190502>\n`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tableName = process.argv[2];

  const records: any = {};
  const q = `select response_message from ${tableName} where request_method = 'POST' and request_url = '/v1.0/login/logintoken' and response_code = 200`;
  (await sql.query(q, {
    type: sql.QueryTypes.SELECT,
  })).forEach(i => {
    try {
      const { _id: id, name, email, token } = JSON.parse(i.response_message);
      const { tenantId, trialExpiry } = <any>jwt.decode(token);
      let record = records[tenantId];
      if (!record) {
        record = records[tenantId] = {
          trialExpiry,
          count: 0,
          breakdown: {},
        };
      }
      record.count++;
      let userStats = record.breakdown[id];
      if (!userStats) {
        userStats = record.breakdown[id] = {
          name,
          email,
          count: 0,
        };
      }
      userStats.count++;
    } catch (e) {
      // ignore
    }
  });
  const output = {
    description:
      'format: <tenant id> -> {count, breakdown: user_id -> {name, email, count}}',
    table_name: tableName,
    long_login_token_stats: records,
  };
  console.log(JSON.stringify(output, null, 2));

  sql.close();
}

main();
