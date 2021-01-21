import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { DocType } from '../rest/model/baseModel';
import { EdgeCert } from '../rest/model/edgeCert';

const USAGE = `\nUsage: node unlockEdgeCertificate.js <edge id>\n`;
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
      if (edgeCert.locked) {
        edgeCert.locked = false;
        const resp = await getDBService().updateDocument(
          edgeCert.tenantId,
          edgeCert.id,
          DocType.EdgeCert,
          edgeCert
        );
        console.log('Unlock edge certificate response:', resp);
      } else {
        console.log('Edge certificate not locked, skip!');
      }
    } else {
      console.log('No edge certificate found, skip!');
    }
  } catch (e) {
    console.log('Unlock edge certificate, caught exception:', e);
  }

  if (sql) {
    sql.close();
  }
}

main();
