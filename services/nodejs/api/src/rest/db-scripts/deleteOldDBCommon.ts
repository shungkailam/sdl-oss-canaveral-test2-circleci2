import { initSequelize } from '../sql-api/baseApi';
import { getDBService, isSQL } from '../db-configurator/dbConfigurator';

async function main() {
  const isSql = isSQL();
  let sql = null;
  if (isSql) {
    sql = initSequelize();
  }
  await getDBService().deleteAllTables();
  if (isSql) {
    await sql.close();
  }
}

main();
