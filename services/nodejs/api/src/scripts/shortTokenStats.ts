import { initSequelize } from '../rest/sql-api/baseApi';

//
// Script to get shortlogintoken stats (request for QR code) across all tenants
//
// Output format:
//   {
//      <tenant id>: {
//         <email>: <count>,
//         ...
//      },
//      ...
//   }
//
const USAGE = `\nUsage: node shortTokenStats.js <table name, e.g., audit_log_20190502>\n`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tableName = process.argv[2];

  const records: any = {};
  const q = `select tenant_id, user_email, count(request_id) from ${tableName} where request_method = 'POST' and request_url = '/v1.0/login/shortlogintoken' and response_code = 200 group by tenant_id, user_email`;
  (await sql.query(q, {
    type: sql.QueryTypes.SELECT,
  })).forEach(i => {
    try {
      const { tenant_id: tenantId, user_email: email, count: sCount } = i;
      const count = Number(sCount);

      let record = records[tenantId];
      if (!record) {
        record = records[tenantId] = {
          count: 0,
          breakdown: {},
        };
      }
      record.count += count;
      record.breakdown[email] = count;
    } catch (e) {
      // ignore
    }
  });
  const output = {
    description: 'format: <tenant id> -> email -> count',
    table_name: tableName,
    short_login_token_stats: records,
  };
  console.log(JSON.stringify(output, null, 2));

  sql.close();
}

main();
