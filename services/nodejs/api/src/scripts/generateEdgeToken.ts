import { isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { createEdgeToken2 } from '../rest/api/edgeApi';

const USAGE = `\nUsage: node generateEdgeToken.js <tenant id> <edge id>\n`;
async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenantId = process.argv[2];
  const edgeId = process.argv[3];

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  console.log('args:', process.argv);

  const { token } = await createEdgeToken2(tenantId, edgeId);
  console.log('edge jwt token:', token);

  if (sql) {
    sql.close();
  }
}

main();
