import { doUpdate } from './common';
import { initSequelize } from '../rest/sql-api/baseApi';

const USAGE = `
Usage: node addDatastreamEndpoints.js go
`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  await doUpdate(
    sql,
    `UPDATE data_stream_model SET end_point = CONCAT('datastream-', id) WHERE destination = 'Cloud' AND cloud_type = 'AWS' AND aws_stream_type = 'S3' AND end_point IS NULL`
  );

  await doUpdate(
    sql,
    `UPDATE data_stream_model SET end_point = CONCAT('datastream-', name) WHERE end_point IS NULL`
  );
  sql.close();
}

main();
