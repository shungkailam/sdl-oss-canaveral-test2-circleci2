import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { DocType } from '../rest/model';

const PROFILE_KEYS = ['enableSSH', 'allowCliSSH', 'privileged'];

const USAGE = `
Usage: node updateTenantProfile.js <tenant id> <profile key> <profile value>

Supported profile keys: enableSSH, allowCliSSH, privileged
Supported values: true, false
`;
async function main() {
  if (process.argv.length < 5) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenantId = process.argv[2];
  const profileKey = process.argv[3];
  const profileValue = process.argv[4];

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  let err = false;

  const doc = <any>(
    await getDBService().getDocument(tenantId, tenantId, DocType.Tenant)
  );

  const foundKey = PROFILE_KEYS.reduce(
    (prev, cur) => cur === profileKey || prev,
    false
  );
  if (!foundKey) {
    console.log('unsupported profile key:', profileKey);
    err = true;
  }

  if (!err) {
    if (profileValue !== 'true' && profileValue !== 'false') {
      console.log('unsupported profile value:', profileValue);
      err = true;
    }
  }

  if (!err) {
    const pv = profileValue === 'true';
    // TODO FIXME: for some reason doc.profile is of type string in
    // getDocument, but in updateDocument we need to pass in object type
    const currProfile = <any>(doc.profile ? JSON.parse(doc.profile) : {});
    currProfile[profileKey] = pv;
    doc.profile = currProfile;
    try {
      await getDBService().updateDocument(
        tenantId,
        tenantId,
        DocType.Tenant,
        doc
      );
    } catch (e) {
      console.log('update tenant failed', e);
      err = true;
    }
  }

  if (sql) {
    sql.close();
  }
  if (err) {
    process.exit(1);
  }
}

main();
