import { isSQL } from '../rest/db-configurator/dbConfigurator';
import { initSequelize } from '../rest/sql-api/baseApi';
import { encryptPassword } from '../rest/util/cryptoUtil';
import { User } from '../rest/model/user';
import { createAdminToken, getUserByEmail } from '../rest/api/userApi';

const USAGE = `\nUsage: node login.js <user email> <user password>\n`;
async function main() {
  if (process.argv.length < 4) {
    console.log(USAGE);
    process.exit(1);
  }
  const email = process.argv[2];
  const password = process.argv[3];

  let sql = null;
  if (isSQL()) {
    sql = initSequelize();
  }

  const user: User = await getUserByEmail(email);
  if (user) {
    if (encryptPassword(password) === user.password) {
      const { token } = await createAdminToken(user);
      console.log('admin jwt token:', token);
    } else {
      console.error(
        `Incorrect password, email: ${email}, password: ${password}`
      );
    }
  }

  if (sql) {
    sql.close();
  }
}

main();
