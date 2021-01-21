import { isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import platformService from '../rest/services/platform.service';
import { doQuery } from './common';

// Get tenant root CA (including unencrypted private key)
// temporary: for manual renew of edge certificates
// may remove in the future

const USAGE = `\nUsage: node getTenantRootCA.js <tenant id>\n`;
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenantId = process.argv[2];

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  try {
    const certs = await doQuery(
      sql,
      `SELECT * FROM tenant_rootca_model WHERE tenant_id = '${tenantId}'`
    );

    if (certs) {
      // decrypt the private key
      const cert = certs[0];
      cert.private_key = await platformService
        .getKeyService()
        .tenantDecrypt(cert.private_key, cert.aws_data_key);
      console.log('Found tenant cert:', cert);
    } else {
      console.log('Failed to find tenant cert?');
    }
  } catch (e) {
    console.log('get tenant certificate, caught exception:', e);
  }

  if (sql) {
    sql.close();
  }
}

main();
