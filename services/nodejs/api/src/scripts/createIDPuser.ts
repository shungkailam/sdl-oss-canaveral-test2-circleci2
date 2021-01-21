/**
 * This is supposed to be used for migrating existing tenant users which are maintained by Andy Daniel.
 * This script creates/updates the user with "iot-admin" role. A tenant ID is created in my.nutanix.com (or demo or stage)
 * that is mapped as external ID to the already existing tenant ID in XI IoT if the email already exists in XI IoT system.
 * Otherwise, the a new tenant ID is created that is later used to map any user belonging to the same external ID.
 * For a new user, an email will be delivered to set the password.
 */

import { createAxios } from '../generator/api';
import * as jwt from 'jsonwebtoken';

const PRIVATE_CERT_DEV = `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`;

const PRIVATE_CERT_STAGE = `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`;

const PRIVATE_CERT_PROD = `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`;

const IDENTITY_USER_PATH = `/api/v2/identity/users`;
const USAGE = `\nUsage: node createIDPUser.js <email> <first name> <last name>\n`;

function getMyNutanixInfo(): any {
  const idpUrl = process.env.OAUTH_IDP || 'https://idp-dev.nutanix.com';
  if (idpUrl.indexOf('://idp.') > 0) {
    return {
      url: 'https://my.nutanix.com',
      iss: '1eaf1492-cbb7-476f-b13a-9bb765e677dc',
      aud: 'https://my.nutanix.com',
      cert: PRIVATE_CERT_PROD,
    };
  }
  if (idpUrl.indexOf('://idp-stage.') > 0) {
    // iss is same as that of dev
    return {
      url: 'https://stage-my.nutanix.com',
      iss: 'd9422b6f-7704-4483-9346-cdb68eff85c2',
      aud: 'https://stage-my.nutanix.com',
      cert: PRIVATE_CERT_STAGE,
    };
  }
  return {
    url: 'https://demo-my.nutanix.com',
    iss: 'd9422b6f-7704-4483-9346-cdb68eff85c2',
    aud: 'https://demo-my.nutanix.com',
    cert: PRIVATE_CERT_DEV,
  };
}

async function getJWT(myNutanixInfo: any): Promise<string> {
  return new Promise<string>(async (resolve, reject) => {
    try {
      const seconds = Math.floor(new Date().getTime() / 1000);
      const claims = {
        iss: myNutanixInfo.iss,
        aud: myNutanixInfo.aud,
        iat: seconds,
        exp: seconds + 60 * 60,
      };
      jwt.sign(claims, myNutanixInfo.cert, { algorithm: 'RS256' }, function(
        err,
        token
      ) {
        if (err) {
          reject(err);
        } else {
          resolve(token);
        }
      });
    } catch (e) {
      reject(e);
    }
  });
}
async function main() {
  if (process.argv.length < 5) {
    console.log(USAGE);
    process.exit(1);
  }
  const user = {
    email: process.argv[2],
    firstName: process.argv[3],
    lastName: process.argv[4],
    targetId: 'iot',
  };
  const myNutanixInfo = getMyNutanixInfo();
  const token = await getJWT(myNutanixInfo);
  console.log('Got URL: ', myNutanixInfo.url);
  const axios = createAxios(myNutanixInfo.url, token);
  try {
    const resp = await axios.post(IDENTITY_USER_PATH, user);
    console.log('IDP response:', resp.status, resp.data);
    if (resp.status === 201 || resp.status === 200) {
      process.exit(0);
    }
    process.exit(resp.status);
  } catch (error) {
    console.log(error);
    process.exit(1);
  }
}

main();
