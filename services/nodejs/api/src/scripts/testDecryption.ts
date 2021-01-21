import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';
import platformService from '../rest/services/platform.service';

//
// This script helps test all data encrypted in DB can be decrypted properly.
//
const USAGE = `
Usage: node testDecryption.js go

`;

const VERBOSE = false;

const keyService = platformService.getKeyService();

async function verifyPK(edge, token) {
  // TODO - handle case where pk was not initially encrypted
  const pkEnc = edge.private_key;
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
  console.log(`done processing edge ${edge.id}...`);
  return pk === pk2;
}

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  // generate tenant token will use KMS master key
  const tt = await platformService.getKeyService().genTenantToken();
  if (VERBOSE) {
    console.log('generated tenant token:', tt);
  }

  const tenantTokenMap = (await doQuery(
    sql,
    `select id, token from tenant_model`
  )).reduce((acc, tenant) => {
    acc[tenant.id] = tenant.token;
    return acc;
  }, {});

  const vpks = await Promise.all(
    (await doQuery(
      sql,
      `select id, tenant_id, edge_id, private_key from edge_cert_model`
    )).map(edge => verifyPK(edge, tenantTokenMap[edge.tenant_id]))
  );
  if (vpks.some(x => !x)) {
    throw Error('Failed: private key decryption mismatch');
  }

  // cloud creds
  const ccs = await doQuery(sql, `select * from cloud_creds_model`);
  const dfs = await Promise.all(
    ccs.map(async cc => {
      let ok = true;
      if (cc.type === 'AWS') {
        if (cc.iflag_encrypted) {
          if (VERBOSE) {
            console.log(
              'encrypted AWS of type ' + typeof cc.aws_credential,
              cc.aws_credential
            );
          }
          try {
            const token = tenantTokenMap[cc.tenant_id];
            const secret = JSON.parse(cc.aws_credential).secret;
            if (VERBOSE) {
              console.log(
                `decrypting AWS secret ${secret} using token ${token}`
              );
            }
            const awsCreds = await keyService.tenantDecrypt(secret, token);
            if (VERBOSE) {
              console.log('decrypting of encrypted AWS gives:', awsCreds);
            }
          } catch (e) {
            console.log('failed to decrypt AWS', cc.aws_credential);
            ok = false;
          }
        } else {
          if (VERBOSE) {
            console.log('unencrypted AWS', cc.aws_credential);
          }
        }
      } else if (cc.type === 'GCP') {
        if (cc.iflag_encrypted) {
          if (VERBOSE) {
            console.log(
              'encrypted GCP of type ' + typeof cc.gcp_credential,
              cc.gcp_credential
            );
          }
          try {
            const token = tenantTokenMap[cc.tenant_id];
            const pk = JSON.parse(cc.gcp_credential).private_key;
            if (VERBOSE) {
              console.log(`decrypting GCP pk ${pk} using token ${token}`);
            }
            const gcpCreds = await keyService.tenantDecrypt(pk, token);
            if (VERBOSE) {
              console.log('decrypting of encrypted GCP gives:', gcpCreds);
            }
          } catch (e) {
            console.log('failed to decrypt GCP', cc.gcp_credential);
            ok = false;
          }
        } else {
          if (VERBOSE) {
            console.log('unencrypted GCP', cc.gcp_credential);
          }
        }
      } else {
        if (VERBOSE) {
          console.log('unknown cloud creds type: ' + cc.type, cc);
        }
      }
      console.log(`done processing cloud profile ${cc.id}...`);
      return ok;
    })
  );
  if (dfs.some(x => !x)) {
    throw Error('Failed: some cloud credential decryption failed');
  }
  console.log(
    'decrypt of edge private key and cloud profiles all done successfully'
  );

  sql.close();
}

main();
