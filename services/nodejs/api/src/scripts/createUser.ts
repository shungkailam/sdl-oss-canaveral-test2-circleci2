import { getDBService, isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { encryptUserPassword } from '../rest/util/cryptoUtil';
import { User, UserRoleType } from '../rest/model/user';
import { DocType } from '../rest/model/baseModel';
import { getUserByEmail } from '../rest/api/userApi';

const USAGE = `\nUsage: node createUser.js <tenant id> <user name> <user email> <user password>\n`;
async function main() {
  if (process.argv.length < 6) {
    console.log(USAGE);
    process.exit(1);
  }
  const tenantId = process.argv[2];
  const user: User = {
    tenantId,
    name: process.argv[3],
    email: process.argv[4],
    password: process.argv[5],
    role: UserRoleType.INFRA_ADMIN,
    version: 0,
  };
  encryptUserPassword(user);

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  try {
    const user2: User = await getUserByEmail(user.email);
    if (user2) {
      console.log(`User with email ${user.email} already exists.`);
      process.exit(1);
    }

    const doc = await getDBService().createDocument(
      tenantId,
      DocType.User,
      user
    );
    console.log('create user returns:', doc);
  } catch (e) {
    console.log(e);
    process.exit(1);
  }
  if (sql) {
    sql.close();
  }
}

main();
