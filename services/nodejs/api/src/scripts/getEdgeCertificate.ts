import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { DocType } from '../rest/model/baseModel';
import { EdgeCert } from '../rest/model/edgeCert';
import { Tenant } from '../rest/model/tenant';
import platformService from '../rest/services/platform.service';

// Get edge certificate (including unencrypted private key)
// mainly for testing purpose, may remove in the future

const USAGE = `\nUsage: node getEdgeCertificate.js <edge id>\n`;
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }
  const edgeId = process.argv[2];

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  try {
    const edgeCert: EdgeCert = await getDBService().findOneDocument<EdgeCert>(
      '',
      { edgeId },
      DocType.EdgeCert
    );
    if (edgeCert) {
      const tenant = await getDBService().findOneDocument<Tenant>(
        '',
        { id: edgeCert.tenantId },
        DocType.Tenant
      );
      if (tenant) {
        // decrypt the private key
        edgeCert.privateKey = await platformService
          .getKeyService()
          .tenantDecrypt(edgeCert.privateKey, tenant.token);
        edgeCert.clientPrivateKey = await platformService
          .getKeyService()
          .tenantDecrypt(edgeCert.clientPrivateKey, tenant.token);
        edgeCert.edgePrivateKey = await platformService
          .getKeyService()
          .tenantDecrypt(edgeCert.edgePrivateKey, tenant.token);
        if (
          edgeCert.certificate !== edgeCert.edgeCertificate ||
          edgeCert.privateKey !== edgeCert.edgePrivateKey
        ) {
          console.log('>>> edgeCert and Cert differs!');
        }
        console.log('Found edge cert:', edgeCert);
      } else {
        console.log('Failed to find tenant for edge cert?');
      }
    } else {
      console.log('No edge certificate found!');
    }
  } catch (e) {
    console.log('get edge certificate, caught exception:', e);
  }

  if (sql) {
    sql.close();
  }
}

main();
