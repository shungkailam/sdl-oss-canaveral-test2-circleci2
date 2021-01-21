import * as childProcess from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

var sql = null;

const USAGE = `\nUsage: node db-tool.js <updateSQL | updateCountSQL <value> | update | updateCount <value>>\n\n
update Updates database to current version.\n
updateCount <value> Applies the next <value> change sets.\n
updateSQL Writes SQL to update database to current version to STDOUT.\n
updateCountSQL <value> Writes SQL to apply the next <value> change sets to STDOUT.\n\n
For more details, refer http://www.liquibase.org/documentation/command_line.html\n`;

const SQL_HOST = process.env.SQL_HOST || 'localhost';
const SQL_PORT = process.env.SQL_PORT || 5432;
const SQL_DB = process.env.SQL_DB || 'sherlock';
const SQL_DIALECT = process.env.SQL_DIALECT || 'postgresql';
const SQL_USER = process.env.SQL_USER || 'root';
const SQL_PASSWORD = process.env.SQL_PASSWORD || 'sherlock';
const SQL_SCRIPT = process.env.SQL_SCRIPT || 'sql-script.sql';

async function execute(command, options): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    try {
      const response = childProcess.execSync(command, options);
      console.log(response.toString());
      resolve();
    } catch (error) {
      console.log(error);
      // print stdout to help debugging
      if (error.stdout && error.stdout.toString) {
        console.log('>>> stdout:', error.stdout.toString('utf8'));
      }
      reject();
    }
  });
}

function extension(element) {
  return path.extname(element) === '.jar';
}

// main function
// declare as async so we can use ES7 async/await
async function main() {
  if (process.argv.length < 3 || process.argv[2] == 'help') {
    console.log(USAGE);
    process.exit(1);
  }
  const command = process.argv.splice(2).join(' ');
  const sqlScript = path.resolve(__dirname, SQL_SCRIPT);
  const jarPath = path.resolve(__dirname, '../../lib');
  const user = `${SQL_USER}`.replace(/\$/, '\\$');
  const password = `${SQL_PASSWORD}`.replace(/\$/, '\\$');
  console.log('SQL script:', sqlScript);
  console.log('Jar file path:', jarPath);
  const filePaths = [];
  fs.readdirSync(jarPath)
    .filter(extension)
    .forEach(file => {
      const jarFile = path.join(jarPath, file);
      console.log('Adding jar:', jarFile);
      filePaths.push(jarFile);
    });
  const classpath = filePaths.join(':');
  console.log(classpath);
  const javaCommand = `java -cp "${classpath}" liquibase.integration.commandline.Main --driver=org.postgresql.Driver --changeLogFile=${sqlScript} --url=jdbc:${SQL_DIALECT}://${SQL_HOST}:${SQL_PORT}/${SQL_DB} --username=${user} --password=${password} ${command}`;
  console.log('Running command', javaCommand);
  try {
    await execute(javaCommand, {});
  } catch (e) {
    console.log(e);
    process.exit(1);
  }
}

main();
