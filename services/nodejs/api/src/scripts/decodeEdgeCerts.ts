import { initSequelize } from '../rest/sql-api/baseApi';
import { doQuery } from './common';
const { execSync } = require('child_process');
import * as fs from 'fs';

//
// Script to find all edges that have short-lived certs
//
const USAGE = `\nUsage: node decodeEdgeCerts.js <tenant id>\n`;

async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }

  let sql = initSequelize();

  const tenantId = process.argv[2];

  const certs = (await doQuery(
    sql,
    `SELECT edge_id, edge_certificate FROM edge_cert_model WHERE tenant_id = '${tenantId}'`
  )).map(({ edge_id, edge_certificate }) => ({ edge_id, edge_certificate }));
  console.log('got tenant edge certs:', certs);
  for (let i = 0; i < certs.length; i++) {
    const { edge_id, edge_certificate } = certs[i];
    fs.writeFileSync('/tmp/x', edge_certificate);
    const stdout = execSync(`openssl x509 -in /tmp/x -noout -dates`);
    const tks = stdout
      .toString()
      .split('\n')[1]
      .split(' ');
    const year = tks[tks.length - 2];
    console.log(`edge id: ${edge_id}, cert expiry year: ${year}`);
  }

  sql.close();
}

main();
