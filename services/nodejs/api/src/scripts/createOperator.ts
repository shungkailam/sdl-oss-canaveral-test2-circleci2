import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { encryptUserPassword } from '../rest/util/cryptoUtil';
import { User, UserRoleType } from '../rest/model/user';
import { DocType } from '../rest/model/baseModel';
import { getTenant } from '../scripts/common';

const OPERATOR_TENANT = 'tid-sherlock-operator';

const USAGE = `\nUsage: node createOperator.js <operator name> <operator email> <operator password> <operator role>\n`;
async function main() {
  if (process.argv.length < 5) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  const tenant = await getTenant(sql, OPERATOR_TENANT);
  if (tenant == null) {
    console.log(`Operator tenant ${OPERATOR_TENANT} does not exist`);
  } else {
    let role = UserRoleType.OPERATOR;
    if (process.argv.length > 5) {
      const roleName = process.argv[5].toUpperCase();
      if ((<any>Object).values(UserRoleType).includes(roleName) === false) {
        console.log('Allowed roles are');
        for (let item in UserRoleType) {
          console.log(' ', item);
        }
        console.log(USAGE);
        process.exit(1);
      }
      role = UserRoleType[roleName];
    }
    const user: User = {
      tenantId: OPERATOR_TENANT,
      name: process.argv[2],
      email: process.argv[3],
      password: process.argv[4],
      role: role,
      version: 0,
    };
    encryptUserPassword(user);

    const doc = await getDBService().createDocument(
      OPERATOR_TENANT,
      DocType.User,
      user
    );
    console.log('create user returns:', doc);
  }
  if (sql) {
    sql.close();
  }
}

main();
