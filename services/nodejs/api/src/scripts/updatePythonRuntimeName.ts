import { doUpdate } from './common';
import { initSequelize } from '../rest/sql-api/baseApi';

const USAGE = `
Usage: node updatePythonRuntimeName.js go
`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  await doUpdate(
    sql,
    `UPDATE script_runtime_model SET name = 'Python3 Env' WHERE id like '%_sr-python' and name = 'Python Env'`
  );
  sql.close();
}

main();
