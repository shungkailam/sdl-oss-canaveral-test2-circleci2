import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';
import platformService from '../rest/services/platform.service';
import { getAllEdgeCerts } from '../rest/api/edgeApi';
import { getDBService } from '../rest/db-configurator/dbConfigurator';
import { DocType } from '../rest/model/baseModel';
import { EdgeCert } from '../rest/model/edgeCert';

//
// This script helps check status and fix edge certificate private keys encryption.
// Run the script with op=status will output bad edge certificate entries.
// (Good entries should have private_key, client_private_key, edge_private_key all encrypted)
// If op=status run found some bad entries (marked with P for Plain, unencrypted),
// the script can be used to encrypt such entries (using op=enc, key=pk or epk or cpk
// for privateKey, edgePrivateKey or clientPrivateKey)
// If there are any bad entries marked with X, then data is corrupted for such entries.
// Corrupted data can not be fixed by this script. (One possible fix is to delete such data.)
//
const USAGE = `
Usage: node edgeCertPkUtil.js <op> <args>...

where <op> can be: status | enc | dec
<args> for each <op>:

  status: [full]
  enc: <tenant id> <edge id> <key>
  dec: <tenant id> <edge id> <key>
where <key> is one of:
  pk | epk | cpk
for privateKey, edgePrivateKey or clientPrivateKey
`;

const VERBOSE = false;

const PRIVATE_KEY_PREFIX = '-----BEGIN ';

const keyService = platformService.getKeyService();
const keyNameMap = {
  pk: 'privateKey',
  cpk: 'clientPrivateKey',
  epk: 'edgePrivateKey',
};

function isPlainPK(pk) {
  return pk.indexOf(PRIVATE_KEY_PREFIX) === 0;
}

async function verifyPK(edge, token, pkName) {
  try {
    // TODO - handle case where pk was not initially encrypted
    const pkEnc = edge[pkName];
    const pk = await keyService.tenantDecrypt(pkEnc, token);
    if (VERBOSE) {
      console.log('edge private key:', pk);
    }
    // note: pkEnc and pkEnc2 will not match due to nonce in encryption
    const pkEnc2 = await keyService.tenantEncrypt(pk, token);
    const pk2 = await keyService.tenantDecrypt(pkEnc2, token);
    if (VERBOSE) {
      console.log('edge pk dec/enc match?', pk === pk2);
    }
    if (VERBOSE) {
      console.log(`done processing edge ${edge.id}...`);
    }
    return pk === pk2;
  } catch (e) {
    console.log(`verify ${pkName} for edge ${edge.id} caught exception`, e);
    return false;
  }
}
// P = Plain, E = Encrypted, X = Bad
async function getPKStatus(edge, token, pkName) {
  if (isPlainPK(edge[pkName])) {
    return 'P';
  }
  const ok = await verifyPK(edge, token, pkName);
  return ok ? 'E' : 'X';
}

const kns = ['private_key', 'client_private_key', 'edge_private_key'];

async function edgeToStatus(edge, token) {
  const edgeStatus: any = { ...edge };
  await Promise.all(
    kns.map(async kn => {
      const s = await getPKStatus(edge, token, kn);
      edgeStatus[kn] = s;
    })
  );
  return edgeStatus;
}

function updateEdgeCert(edgeCert: EdgeCert) {
  return getDBService().updateDocument(
    edgeCert.tenantId,
    edgeCert.id,
    DocType.EdgeCert,
    edgeCert
  );
}

async function encryptEdgeCert(edgeCert, token, kn) {
  if (VERBOSE) {
    console.log(
      'encryptEdgeCert { token=' + token + ', key name=' + kn + ', edgeCert=',
      edgeCert
    );
  }
  const pk = edgeCert[kn];
  const pkEnc = await keyService.tenantEncrypt(pk, token);
  edgeCert[kn] = pkEnc;
  if (VERBOSE) {
    console.log('encryptEdgeCert, encrypted ', pk, '\nto\n', pkEnc);
  }
}
async function decryptEdgeCert(edgeCert, token, kn) {
  if (VERBOSE) {
    console.log(
      'decryptEdgeCert { token=' + token + ', key name=' + kn + ', edgeCert=',
      edgeCert
    );
  }
  const pkEnc = edgeCert[kn];
  const pk = await keyService.tenantDecrypt(pkEnc, token);
  edgeCert[kn] = pk;
  if (VERBOSE) {
    console.log('decryptEdgeCert, decrypted ', pkEnc, '\nto\n', pk);
  }
}

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  const op = process.argv[2];
  const isStatus = op === 'status';
  const isEnc = op === 'enc';
  const isDec = op === 'dec';
  if (!isStatus && !isEnc && !isDec) {
    console.log(USAGE);
    process.exit(1);
  }

  let keyName = '';
  if (!isStatus) {
    if (process.argv.length < 6) {
      console.log(USAGE);
      process.exit(1);
    }
    keyName = process.argv[5];
    if (keyName !== 'pk' && keyName !== 'cpk' && keyName !== 'epk') {
      console.log(USAGE);
      process.exit(1);
    }
  }

  let sql = initSequelize();

  const tenantTokenMap = (await doQuery(
    sql,
    `select id, token from tenant_model`
  )).reduce((acc, tenant) => {
    acc[tenant.id] = tenant.token;
    return acc;
  }, {});

  const edges = await doQuery(
    sql,
    `select id, tenant_id, edge_id, private_key, client_private_key, edge_private_key from edge_cert_model`
  );

  if (isStatus) {
    const isFullStatus = process.argv[3] === 'full';

    // verifyPK(edge, tenantTokenMap[edge.tenant_id]))
    const edgesStatus: any[] = await Promise.all(
      edges.map(edge => {
        const token = tenantTokenMap[edge.tenant_id];
        return edgeToStatus(edge, token);
      })
    );
    const edgesStatusMap: any = {};
    const badEdgesStatusMap: any = {};
    edgesStatus.forEach(es => {
      edgesStatusMap[es.id] = es;
      if (
        es.private_key !== 'E' ||
        es.edge_private_key !== 'E' ||
        es.client_private_key !== 'E'
      ) {
        badEdgesStatusMap[es.id] = es;
      }
    });
    if (isFullStatus) {
      console.log(JSON.stringify(edgesStatusMap, null, 2));
    }
    if (Object.keys(badEdgesStatusMap).length !== 0) {
      console.log('\n\n>>> BAD edges:', badEdgesStatusMap);
    } else {
      console.log('\n\n>>> All edges private keys properly encrypted.');
    }
  } else {
    const tenantId = process.argv[3];
    const edgeId = process.argv[4];
    // first get the edge cert
    const edgeCerts = await getAllEdgeCerts(tenantId);
    if (VERBOSE) {
      console.log('Got edge certs:', edgeCerts);
    }
    const edgeCert = edgeCerts.find(ec => ec.edgeId === edgeId);
    if (edgeCert === null) {
      console.log('Error: Edge cert not found');
      process.exit(1);
    }
    if (VERBOSE) {
      console.log('Got edge cert:', edgeCert);
    }
    // then apply the operation
    const token = tenantTokenMap[tenantId];
    const kn = keyNameMap[keyName];
    if (VERBOSE) {
      console.log('Token: ', token);
      console.log('key name:', kn);
    }
    if (isEnc) {
      // encrypt edge cert
      await encryptEdgeCert(edgeCert, token, kn);
    } else {
      // decrypt edge cert
      await decryptEdgeCert(edgeCert, token, kn);
    }
    // then save the edge cert
    await updateEdgeCert(edgeCert);
  }

  sql.close();
}

main();
