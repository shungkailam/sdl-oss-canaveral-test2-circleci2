import { initSequelize } from '../../rest/sql-api/baseApi';
import { getDBService } from '../../rest/db-configurator/dbConfigurator';
import { AwsKmsKeyService } from './keyService';
import { getAllTenants, getAllTenantRootCAs } from '../../rest/api/tenantApi';
import { getAllCloudCreds } from '../../rest/api/cloudCredsApi';
import { getAllEdgeCerts } from '../../rest/api/edgeApi';
import { DocType } from '../../rest/model/baseModel';
import { CloudCreds } from '../../rest/model/cloudCreds';
import { DockerProfile } from '../../rest/model/dockerProfile';
import { Tenant } from '../../rest/model/tenant';
import { TenantRootCA } from '../../rest/model/tenantRootca';
import { EdgeCert } from '../../rest/model/edgeCert';
import { getAllDockerProfiles } from '../../rest/api/dockerProfileApi';

// This script helps to migrate Sherlock cloudmgmt DB data
// from one KMS master key to another.
// This is needed as we move beta from dev account to prod account
// and use different KMS master key for prod account from dev account.
//
// This script should be run inside a cloudmgmt pod.
// It migrates one DB (i.e., one namespace, e.g., sherlock_beta) at a time.
//
// To ensure no REST API access during the migration,
// before running the script, one should delete the corresponding
// ingress resource first so no request can come in during migration.
// After the migration is done one can then re-create the ingress resources.
//
// This script should be run in two passes.
// The first pass performs decryption. The second pass updates the data keys
// using the new kms key and performs encryption using the new data keys.
// pass 1:
//   node dist/scripts/kms-migration/kmsMigration.js dec true
// pass 2:
//   node dist/scripts/kms-migration/kmsMigration.js enc true <new kms key arn>

const DEBUG = false;

const PRIVATE_KEY_PREFIX = '-----BEGIN PRIVATE KEY-----';
const RBAC_TEST_PRIVATE_KEY_PREFIX = 'private_key-';
const RSA_PRIVATE_KEY_PREFIX = '-----BEGIN RSA PRIVATE KEY-----';
const DUMMY_KEY = 'foobar';
const CHECK_BAD_GCP_PRIVATE_KEY = false;

const SMAX = 180;

function truncate(s, max) {
  if (s.length > max) {
    return s.substring(0, max) + '...';
  }
  return s;
}

function updateDockerProfile(tenant: Tenant, dockerProfile: DockerProfile) {
  return getDBService().updateDocument(
    tenant.id,
    dockerProfile.id,
    DocType.DockerProfile,
    dockerProfile
  );
}
function updateCloudCreds(tenant: Tenant, cloudCreds: CloudCreds) {
  return getDBService().updateDocument(
    tenant.id,
    cloudCreds.id,
    DocType.CloudCreds,
    cloudCreds
  );
}
function updateTenant(tenant: Tenant) {
  return getDBService().updateDocument(
    tenant.id,
    tenant.id,
    DocType.Tenant,
    tenant
  );
}
function updateTenantRootCA(tenantRootCA: TenantRootCA) {
  return getDBService().updateDocument(
    tenantRootCA.tenantId,
    tenantRootCA.id,
    DocType.TenantRootCA,
    tenantRootCA
  );
}
function updateEdgeCert(edgeCert: EdgeCert) {
  return getDBService().updateDocument(
    edgeCert.tenantId,
    edgeCert.id,
    DocType.EdgeCert,
    edgeCert
  );
}

async function decryptCloudCred(
  cloudCred: CloudCreds,
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  if (DEBUG) {
    console.log(`\

>>> Found encrypted cloud profile with id ${cloudCred.id}

        `);
  }
  cloudCred.iflagEncrypted = false;
  if (cloudCred.type === 'GCP') {
    if (cloudCred.gcpCredential) {
      const gcpc = JSON.parse(<any>cloudCred.gcpCredential);
      if (gcpc.private_key) {
        const pk = await kmsService.tenantDecrypt(
          gcpc.private_key,
          tenant.token
        );
        console.log(
          `[${cloudCred.id}] decrypted cloud profile gcp pk from ${truncate(
            gcpc.private_key,
            SMAX
          )} to ${truncate(pk, SMAX)}`
        );
        if (CHECK_BAD_GCP_PRIVATE_KEY) {
          if (
            pk.indexOf(PRIVATE_KEY_PREFIX) !== 0 &&
            pk.indexOf(RBAC_TEST_PRIVATE_KEY_PREFIX) !== 0 &&
            pk !== ''
          ) {
            throw Error('Bad private key? ' + pk);
          }
        }
        gcpc.private_key = pk;
        cloudCred.gcpCredential = <any>JSON.stringify(gcpc);
        if (DEBUG) {
          console.log('cc: decrypted GCP:', cloudCred);
        }
      }
    }
  } else if (cloudCred.type === 'AWS') {
    const awsc = JSON.parse(<any>cloudCred.awsCredential);
    if (awsc.secret) {
      const secret = await kmsService.tenantDecrypt(awsc.secret, tenant.token);
      console.log(
        `[${cloudCred.id}] decrypted cloud profile aws secret from ${
          awsc.secret
        }  to ${secret}`
      );
      awsc.secret = secret;
      cloudCred.awsCredential = <any>JSON.stringify(awsc);
      if (DEBUG) {
        console.log('cc: decrypted AWS:', cloudCred);
      }
    }
  } else {
    throw Error('Encrypted non-(GCP, AWS) cloud profile found');
  }
  // now update cloud profile
  if (updateDB) {
    console.log('>>> updating cloud profile:', cloudCred);
    await updateCloudCreds(tenant, cloudCred);
  }
}

async function decryptDockerProfile(
  dockerProfile: DockerProfile,
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  if (DEBUG) {
    console.log(`\

>>> Found encrypted docker profile with id ${dockerProfile.id}

        `);
  }
  dockerProfile.iflagEncrypted = false;
  if (
    dockerProfile.type === 'GCP' ||
    dockerProfile.type === 'ContainerRegistry'
  ) {
    if (dockerProfile.pwd) {
      const pwd = await kmsService.tenantDecrypt(
        dockerProfile.pwd,
        tenant.token
      );
      console.log(
        `[${dockerProfile.id}] decrypted docker profile pwd from ${truncate(
          dockerProfile.pwd,
          SMAX
        )} to ${truncate(pwd, SMAX)}`
      );
      dockerProfile.pwd = pwd;
    }
  } else {
    throw Error(
      'Encrypted non-(GCP, AWS, ContainerRegistry) docker profile found'
    );
  }
  // now update docker profile
  if (updateDB) {
    console.log('>>> updating docker profile:', dockerProfile);
    await updateDockerProfile(tenant, dockerProfile);
  }
}

async function encryptCloudCred(
  cloudCred: CloudCreds,
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  if (DEBUG) {
    console.log(`\

>>> Found non-encrypted cloud profile with id ${cloudCred.id}

        `);
  }

  cloudCred.iflagEncrypted = true;
  if (cloudCred.type === 'GCP') {
    if (cloudCred.gcpCredential) {
      const gcpc = JSON.parse(<any>cloudCred.gcpCredential);
      if (gcpc.private_key) {
        if (CHECK_BAD_GCP_PRIVATE_KEY) {
          if (
            gcpc.private_key.indexOf(PRIVATE_KEY_PREFIX) !== 0 &&
            gcpc.private_key.indexOf(RBAC_TEST_PRIVATE_KEY_PREFIX) !== 0
          ) {
            throw Error('Bad private key? ' + gcpc.private_key);
          }
        }
        const pk = await kmsService.tenantEncrypt(
          gcpc.private_key,
          tenant.token
        );
        console.log(
          `[${cloudCred.id}] encrypted gcp pk from ${truncate(
            gcpc.private_key,
            SMAX
          )} to ${truncate(pk, SMAX)}`
        );
        gcpc.private_key = pk;
        cloudCred.gcpCredential = <any>JSON.stringify(gcpc);
        if (DEBUG) {
          console.log('cc: decrypted GCP:', cloudCred);
        }
      }
    }
  } else if (cloudCred.type === 'AWS') {
    if (cloudCred.awsCredential) {
      const awsc = JSON.parse(<any>cloudCred.awsCredential);
      if (awsc.secret) {
        const secret = await kmsService.tenantEncrypt(
          awsc.secret,
          tenant.token
        );
        console.log(
          `[${cloudCred.id}] encrypted aws secret from ${
            awsc.secret
          }  to ${secret}`
        );
        awsc.secret = secret;
        cloudCred.awsCredential = <any>JSON.stringify(awsc);
        if (DEBUG) {
          console.log('cc: decrypted AWS:', cloudCred);
        }
      }
    }
  } else {
    throw Error('Encrypted non-(GCP, AWS) cloud profile found');
  }
  // now update cloud profile
  if (updateDB) {
    console.log('>>> updating cloud profile:', cloudCred);
    await updateCloudCreds(tenant, cloudCred);
  }
}

async function encryptDockerProfile(
  dockerProfile: DockerProfile,
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  if (DEBUG) {
    console.log(`\

>>> Found non-encrypted docker profile with id ${dockerProfile.id}

        `);
  }

  dockerProfile.iflagEncrypted = true;
  if (
    dockerProfile.type === 'GCP' ||
    dockerProfile.type === 'ContainerRegistry'
  ) {
    if (dockerProfile.pwd) {
      const pwd = await kmsService.tenantEncrypt(
        dockerProfile.pwd,
        tenant.token
      );
      console.log(
        `[${dockerProfile.id}] encrypted pwd from ${truncate(
          dockerProfile.pwd,
          SMAX
        )} to ${truncate(pwd, SMAX)}`
      );
      dockerProfile.pwd = pwd;
    }
  } else {
    throw Error(
      'Encrypted non-(GCP, AWS, ContainerRegistry) docker profile found'
    );
  }
  // now update docker profile
  if (updateDB) {
    console.log('>>> updating docker profile:', dockerProfile);
    await updateDockerProfile(tenant, dockerProfile);
  }
}

async function decryptEdgeCert(
  edgeCert: EdgeCert,
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  const pkDummy = edgeCert.privateKey === DUMMY_KEY;
  const pkPlain = edgeCert.privateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0;
  const pkEdgePlain =
    edgeCert.edgePrivateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0;
  const pkClientPlain =
    edgeCert.clientPrivateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0;

  if (pkEdgePlain && pkClientPlain) {
    if (pkPlain || pkDummy) {
      return;
    } else {
      // bad data
      console.error('bad edge cert data:', edgeCert);
      throw Error('bad edge cert data');
    }
  } else if (pkEdgePlain || pkClientPlain) {
    // bad data
    console.error('bad edge cert data:', edgeCert);
    throw Error('bad edge cert data');
  }

  let pk = edgeCert.privateKey;
  if (!pkDummy && !pkPlain) {
    pk = await kmsService.tenantDecrypt(edgeCert.privateKey, tenant.token);
  }
  const pkClient = await kmsService.tenantDecrypt(
    edgeCert.clientPrivateKey,
    tenant.token
  );
  const pkEdge = await kmsService.tenantDecrypt(
    edgeCert.edgePrivateKey,
    tenant.token
  );
  console.log(
    `[${edgeCert.id}] decrypted privateKey from ${truncate(
      edgeCert.privateKey,
      SMAX
    )} to ${truncate(pk, SMAX)}`
  );
  console.log(
    `[${edgeCert.id}] decrypted edgePrivateKey from ${truncate(
      edgeCert.edgePrivateKey,
      SMAX
    )} to ${truncate(pkEdge, SMAX)}`
  );
  console.log(
    `[${edgeCert.id}] decrypted clientPrivateKey from ${truncate(
      edgeCert.clientPrivateKey,
      SMAX
    )} to ${truncate(pkClient, SMAX)}`
  );
  edgeCert.privateKey = pk;
  edgeCert.clientPrivateKey = pkClient;
  edgeCert.edgePrivateKey = pkEdge;
  if (updateDB) {
    await updateEdgeCert(edgeCert);
  }
}

async function encryptEdgeCert(
  edgeCert: EdgeCert,
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  const pkDummy = edgeCert.privateKey === DUMMY_KEY;
  const pkPlain = edgeCert.privateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0;
  const pkEdgePlain =
    edgeCert.edgePrivateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0;
  const pkClientPlain =
    edgeCert.clientPrivateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0;
  if (!pkPlain && !pkEdgePlain && !pkClientPlain) {
    // verify proper encryption
    let pkD = RSA_PRIVATE_KEY_PREFIX;
    if (!pkDummy) {
      pkD = await kmsService.tenantDecrypt(edgeCert.privateKey, tenant.token);
    }
    const pkClientD = await kmsService.tenantDecrypt(
      edgeCert.clientPrivateKey,
      tenant.token
    );
    const pkEdgeD = await kmsService.tenantDecrypt(
      edgeCert.edgePrivateKey,
      tenant.token
    );
    if (
      pkD.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0 &&
      pkClientD.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0 &&
      pkEdgeD.indexOf(RSA_PRIVATE_KEY_PREFIX) === 0
    ) {
      // ok
      return;
    } else {
      // bad data
      console.error('bad edge cert data:', edgeCert);
      throw Error('bad edge cert data');
    }
  } else if (!(pkPlain || pkDummy) || !pkEdgePlain || !pkClientPlain) {
    // bad data
    console.error('bad edge cert data:', edgeCert);
    throw Error('bad edge cert data');
  }

  let pk = edgeCert.privateKey;
  if (!pkDummy) {
    pk = await kmsService.tenantEncrypt(edgeCert.privateKey, tenant.token);
  }
  const pkClient = await kmsService.tenantEncrypt(
    edgeCert.clientPrivateKey,
    tenant.token
  );
  const pkEdge = await kmsService.tenantEncrypt(
    edgeCert.edgePrivateKey,
    tenant.token
  );
  console.log(
    `[${edgeCert.id}] encrypted privateKey from ${truncate(
      edgeCert.privateKey,
      SMAX
    )} to ${truncate(pk, SMAX)}`
  );
  console.log(
    `[${edgeCert.id}] encrypted edgePrivateKey from ${truncate(
      edgeCert.edgePrivateKey,
      SMAX
    )} to ${truncate(pkEdge, SMAX)}`
  );
  console.log(
    `[${edgeCert.id}] encrypted clientPrivateKey from ${truncate(
      edgeCert.clientPrivateKey,
      SMAX
    )} to ${truncate(pkClient, SMAX)}`
  );
  edgeCert.privateKey = pk;
  edgeCert.clientPrivateKey = pkClient;
  edgeCert.edgePrivateKey = pkEdge;
  if (updateDB) {
    await updateEdgeCert(edgeCert);
  }
}

function asyncThrottle(fn) {
  return async function foo(items: any[], ...args) {
    if (items.length) {
      const item = items.shift();
      await fn.apply(null, [item].concat(args));
      await foo.apply(null, [items].concat(args));
    }
  };
}

const encryptTenantDockerProfilesRecursive = asyncThrottle(
  encryptDockerProfile
);
const encryptTenantCloudCredsRecursive = asyncThrottle(encryptCloudCred);
const encryptEdgeCertsRecursive = asyncThrottle(encryptEdgeCert);

const decryptTenantDockerProfilesRecursive = asyncThrottle(
  decryptDockerProfile
);
const decryptTenantCloudCredsRecursive = asyncThrottle(decryptCloudCred);
const decryptEdgeCertsRecursive = asyncThrottle(decryptEdgeCert);

async function decryptTenantData(
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  console.log(`
====================================================================
=   handling tenant ${tenant.id}
====================================================================`);
  // cloud profile
  // get all cloud profiles currently encrypted
  // for descryption it is ok if not all cloud profiles are currently encrypted
  const cloudCreds = (await getAllCloudCreds(tenant.id)).filter(
    cloudCred => cloudCred.iflagEncrypted
  );
  if (cloudCreds.length) {
    console.log(
      `currently encrypted cloud profiles count: ${cloudCreds.length}`
    );
    if (DEBUG) {
      console.log(
        'currently encrypted cloud profiles for tenant id ' + tenant.id,
        cloudCreds
      );
    }
    await decryptTenantCloudCredsRecursive(
      cloudCreds,
      tenant,
      kmsService,
      updateDB
    );
  } else {
    console.log('decrypt cloud profile: nothing to do');
  }

  // docker profiles
  // get all non-AWS docker profiles currently encrypted
  // we skip AWS docker profiles as their encryption is not yet supported
  const dockerProfiles = (await getAllDockerProfiles(tenant.id)).filter(
    dockerProfile => dockerProfile.iflagEncrypted && dockerProfile.type != 'AWS'
  );
  if (dockerProfiles.length) {
    console.log(
      `currently encrypted docker profiles count: ${dockerProfiles.length}`
    );
    await decryptTenantDockerProfilesRecursive(
      dockerProfiles,
      tenant,
      kmsService,
      updateDB
    );
  } else {
    console.log('decrypt docker profile: nothing to do');
  }

  // edge certs
  const edgeCerts = await getAllEdgeCerts(tenant.id);
  if (edgeCerts.length) {
    await decryptEdgeCertsRecursive(edgeCerts, tenant, kmsService, updateDB);
  } else {
    console.log('decrypt edge certs: nothing to do');
  }
}

async function encryptTenantData(
  tenant: Tenant,
  kmsService: AwsKmsKeyService,
  updateDB: boolean,
  kmsKeyId: string
) {
  console.log(`
====================================================================
=   handling tenant ${tenant.id}
====================================================================`);
  // update tenant token
  if (kmsKeyId) {
    const token = await kmsService.genTenantToken(kmsKeyId);
    tenant.token = token;
    await updateTenant(tenant);
  }

  // cloud profile
  const cloudCreds = (await getAllCloudCreds(tenant.id)).filter(
    cloudCred => !cloudCred.iflagEncrypted
  );
  if (cloudCreds.length) {
    console.log(`plain cloud profile count: ${cloudCreds.length}`);
    if (DEBUG) {
      console.log('cloud creds for tenant id ' + tenant.id, cloudCreds);
    }
    // cloud profile - must not have any already encrypted
    const encryptedCloudCreds = (await getAllCloudCreds(tenant.id)).filter(
      cloudCred => cloudCred.iflagEncrypted
    );
    if (encryptedCloudCreds.length) {
      throw Error(
        `encryptTenantData: pre-condition failed: expect encrypted cloud profile count to be 0, found ${
          encryptedCloudCreds.length
        }`
      );
    }
    await encryptTenantCloudCredsRecursive(
      cloudCreds,
      tenant,
      kmsService,
      updateDB
    );
  } else {
    console.log('encrypt cloud profile: nothing to do');
  }

  // docker profile
  const dockerProfiles = (await getAllDockerProfiles(tenant.id)).filter(
    dockerProfile =>
      !dockerProfile.iflagEncrypted && dockerProfile.type != 'AWS'
  );
  if (dockerProfiles.length) {
    console.log(`plain docker profile count: ${dockerProfiles.length}`);
    if (DEBUG) {
      console.log('docker profiles for tenant id ' + tenant.id, dockerProfiles);
    }
    // docker profile - must not have any already encrypted
    const encryptedDockerProfiles = (await getAllDockerProfiles(
      tenant.id
    )).filter(
      dockerProfile =>
        dockerProfile.iflagEncrypted && dockerProfile.type != 'AWS'
    );
    if (encryptedDockerProfiles.length) {
      throw Error(
        `encryptTenantData: pre-condition failed: expect encrypted docker profile count to be 0, found ${
          encryptedDockerProfiles.length
        }`
      );
    }
    await encryptTenantDockerProfilesRecursive(
      dockerProfiles,
      tenant,
      kmsService,
      updateDB
    );
  } else {
    console.log('encrypt docker profile: nothing to do');
  }

  // edge certs
  const edgeCerts = await getAllEdgeCerts(tenant.id);
  if (edgeCerts.length) {
    await encryptEdgeCertsRecursive(edgeCerts, tenant, kmsService, updateDB);
  } else {
    console.log('encrypt edge certs: nothing to do');
  }
}

const encryptTenantsRecursive = asyncThrottle(encryptTenantData);
const decryptTenantsRecursive = asyncThrottle(decryptTenantData);

async function encryptTenantRootCA(
  tenantRootCA: TenantRootCA,
  kmsService: AwsKmsKeyService,
  updateDB: boolean,
  kmsKeyId: string
) {
  console.log(`
  ====================================================================
  =   encrypt rootca for tenant ${tenantRootCA.tenantId}
  ====================================================================`);
  if (tenantRootCA.privateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) !== 0) {
    // see if already properly encrypted
    const pk = await kmsService.tenantDecrypt(
      tenantRootCA.privateKey,
      tenantRootCA.awsDataKey
    );
    if (pk.indexOf(RSA_PRIVATE_KEY_PREFIX) !== 0) {
      throw Error(
        `Bad rootca? id=${tenantRootCA.id}, pk=${truncate(
          tenantRootCA.privateKey,
          SMAX
        )}, decrypted pk=${truncate(pk, SMAX)}`
      );
    }
    if (!kmsKeyId) {
      return;
    }
    tenantRootCA.privateKey = pk;
  }
  // update data key if new kms key is provided
  if (kmsKeyId) {
    tenantRootCA.awsDataKey = await kmsService.genTenantToken(kmsKeyId);
  }
  const pk = await kmsService.tenantEncrypt(
    tenantRootCA.privateKey,
    tenantRootCA.awsDataKey
  );
  console.log(
    `encryted rootca private key from ${truncate(
      tenantRootCA.privateKey,
      SMAX
    )} to ${truncate(pk, SMAX)}`
  );
  tenantRootCA.privateKey = pk;
  if (updateDB) {
    await updateTenantRootCA(tenantRootCA);
  }
}
async function decryptTenantRootCA(
  tenantRootCA: TenantRootCA,
  kmsService: AwsKmsKeyService,
  updateDB: boolean
) {
  console.log(`
  ====================================================================
  =   decrypt rootca for tenant ${tenantRootCA.tenantId}
  ====================================================================`);
  if (tenantRootCA.privateKey.indexOf(RSA_PRIVATE_KEY_PREFIX) !== 0) {
    const pk = await kmsService.tenantDecrypt(
      tenantRootCA.privateKey,
      tenantRootCA.awsDataKey
    );
    console.log(
      `decryted rootca private key from ${truncate(
        tenantRootCA.privateKey,
        SMAX
      )} to ${truncate(pk, SMAX)}`
    );
    tenantRootCA.privateKey = pk;
    if (updateDB) {
      await updateTenantRootCA(tenantRootCA);
    }
  } else {
    console.log('decrypt rootca: nothing to do');
  }
}
const encryptTenantRootCARecursive = asyncThrottle(encryptTenantRootCA);
const decryptTenantRootCARecursive = asyncThrottle(decryptTenantRootCA);

const USAGE = `
Usage: node kmsMigration.js <enc | dec> <update DB> [<new kms master key arn>]
where
       enc = encryption
       dec = decryption
       update DB = true or false, if true, will update the DB
       new kms key = only relevant for enc, if supplied will update tenant with new token
`;

async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  let exitCode = 0;

  const mode = process.argv[2];
  const updateDB = process.argv[3] === 'true';
  const newKMSKeyId = process.argv[4];

  const kmsService = new AwsKmsKeyService();

  let sql = initSequelize();

  try {
    const tenants = await getAllTenants();
    const tenantRootCAs = await getAllTenantRootCAs();
    console.log(
      `Processing ${tenants.length} tenants, ${
        tenantRootCAs.length
      } tenant RootCAs`
    );
    if (DEBUG) {
      console.log('tenants:', tenants);
    }
    if (mode === 'enc') {
      await encryptTenantsRecursive(tenants, kmsService, updateDB, newKMSKeyId);
      await encryptTenantRootCARecursive(
        tenantRootCAs,
        kmsService,
        updateDB,
        newKMSKeyId
      );
    } else if (mode === 'dec') {
      await decryptTenantsRecursive(tenants, kmsService, updateDB);
      await decryptTenantRootCARecursive(tenantRootCAs, kmsService, updateDB);
    } else {
      console.error(`Unknown mode: '${mode}', must be 'enc' or 'dec'`);
    }
  } catch (e) {
    console.error('>>> caught exception:', e);
    exitCode = 500;
  }

  sql.close();

  process.exit(exitCode);
}

main();
