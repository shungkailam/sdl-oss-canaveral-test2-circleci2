import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import platformService from '../rest/services/platform.service';
import { DocType } from '../rest/model';

const USAGE = `\nUsage: node createDomain.js <tenant id> <domain name>\n`;
async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  const domain = {
    tenantId: process.argv[2],
    name: process.argv[3],
    token: await platformService.getKeyService().genTenantToken(),
    version: 0,
    description: '',
  };

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  const doc = await getDBService().createDocument(
    domain.tenantId,
    DocType.Domain,
    domain
  );
  console.log('create domain returns:', doc);

  if (sql) {
    sql.close();
  }
}

main();
