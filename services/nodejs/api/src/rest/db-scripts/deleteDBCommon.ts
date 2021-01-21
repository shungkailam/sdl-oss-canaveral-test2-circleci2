import { initSequelize } from '../sql-api/baseApi';
import { isSQL } from '../db-configurator/dbConfigurator';

const DESTROY_SQL_SCRIPTS = `
drop table if exists log_model cascade;
drop table if exists script_model cascade;
drop table if exists script_runtime_model cascade;
drop table if exists sensor_model cascade;
drop table if exists user_model cascade;
drop table if exists edge_cert_model cascade;
drop table if exists edge_model cascade;
drop table if exists data_stream_model cascade;
drop table if exists data_stream_origin_model cascade;
drop table if exists application_model cascade;
drop table if exists application_status_model cascade;
drop table if exists data_source_model cascade;
drop table if exists data_source_field_selector_model cascade;
drop table if exists data_source_field_model cascade;
drop table if exists cloud_creds_model cascade;
drop table if exists docker_profile_model cascade;
drop table if exists category_value_model cascade;
drop table if exists category_model cascade;
drop table if exists tenant_model cascade;
drop table if exists databasechangeloglock cascade;
drop table if exists databasechangelog cascade;
`;

async function main() {
  const isSql = isSQL();
  let sql = null;
  if (isSql) {
    sql = initSequelize();
  }
  try {
    await sql.query(DESTROY_SQL_SCRIPTS);
  } finally {
    if (isSql) {
      await sql.close();
    }
  }
}

main();
