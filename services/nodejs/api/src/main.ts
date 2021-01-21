import * as sjcl from 'sjcl';
import * as sha256 from 'sha256';
import {
  DocType,
  DocTypes,
  CategoryKeys,
  DataSourceKeys,
  DataStreamKeys,
  EdgeKeys,
  ScriptKeys,
  SensorKeys,
  TenantKeys,
  UserKeys,
} from './rest/model/index';

import { getDBService, isSQL } from './rest/db-configurator/dbConfigurator';
import { initSequelize } from './rest/sql-api/baseApi';
import { getAllEdges, getAllDataSources } from './rest/api/index';
import AxiosLib from 'axios';
import { getEdgeHandleToken } from './rest/util/cryptoUtil';
import * as crypto2 from 'crypto2';
import * as jwt from 'jsonwebtoken';
import * as AWS from 'aws-sdk';
import { AnalyticsAndOperator } from 'aws-sdk/clients/s3';
import cryptoKeyService from './rest/services/impl/key.service.crypto';
import { AwsKeyService } from './rest/services/impl/key.service.aws';
import { KeyService } from './rest/services/key.service';
import * as uuidv4 from 'uuid/v4';

import * as fs from 'fs';
import * as forge from 'node-forge';
import * as NodeRSA from 'node-rsa';

function getModelKeys(type) {
  switch (type) {
    case DocType.Category:
      return CategoryKeys;
    case DocType.DataSource:
      return DataSourceKeys;
    case DocType.DataStream:
      return DataStreamKeys;
    case DocType.Edge:
      return EdgeKeys;
    case DocType.Script:
      return ScriptKeys;
    case DocType.Sensor:
      return SensorKeys;
    case DocType.Tenant:
      return TenantKeys;
    case DocType.User:
      return UserKeys;
    default:
      return [];
  }
}
// const sha = new sjcl.hash.sha256();
// sha.update('apex');

function shaWithSalt(input, salt) {
  return sha256(input + salt);
}

function testSha() {
  console.log('sha of apex: ', sjcl.hash.sha256.hash('apex'));
  console.log('sha256 of apex: ', sha256('apex'));
  const salt = sha256('sherlock');
  console.log('sha with salt of apex: ', shaWithSalt('apex', salt));
}

async function testCreateTenant() {
  const tenant = {
    id: process.argv[2] || 'test-create-tenant-id',
    name: process.argv[3] || 'Test Create Tenant',
    token: process.argv[4] || 'test-create-tenant-token',
  };
  const doc = await getDBService().createTenant('', tenant);
  console.log('create tenant returns:', doc);
}

async function testApiPerf() {
  const startTime = Date.now();
  const ps = await Promise.all(
    Array(10)
      .fill(0)
      .map(x => getAllDataSources('tenant-id-waldot'))
  );
  console.log(`got edges in ${Date.now() - startTime}ms`, ps);
}

const ENDPOINT = process.env.ENDPOINT || 'http://localhost:3000';
const API_PATH = process.env.API_PATH || '/v1/edges';
const axiosInstance = getAxios(ENDPOINT);

function getAxios(endpoint) {
  return AxiosLib.create({
    baseURL: endpoint,
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: 60000,
  });
}

// TODO - FIXME - only works for flat promise arrays
// nested promises won't be sequential
function seqPromiseAll(promises, acc = [], idx = 0) {
  if (!promises || !promises.length || idx >= promises.length) {
    return acc;
  }
  const promise = promises[idx];
  return promise.then(r => {
    acc.push(r);
    return seqPromiseAll(promises, acc, idx + 1);
  });
}

// const ENDPOINTS = ['https://sherlockntnx.com', 'https://ntnxsherlock.com'];
// const ENDPOINTS = ['https://sherlockntnx.com'];
const ENDPOINTS = ['https://sherlockntnx.com', 'http://localhost:3000'];
const API_PATHS = [
  'datasources',
  'edges',
  'categories',
  'scripts',
  'datastreams',
  'sensors',
];
// const ITERATIONS = [1, 10, 100];
const ITERATIONS = [1, 3];

function logTime(endpoint, path, idx, tobj) {
  // TODO
  console.log(
    `logTime: endpoint=${endpoint}, path=${path}, idx=${idx}, time=${tobj.time}`
  );
}
async function oneApiCall(axios, endpoint, path, idx) {
  const st = Date.now();
  try {
    const data = await axios.get(path).then(res => res.data);
    logTime(endpoint, path, idx, {
      time: Date.now() - st,
    });
  } catch (e) {
    console.log('>>> error:', e);
    logTime(endpoint, path, idx, {
      time: Date.now() - st,
      error: true,
    });
  }
}

async function allApiCalls() {
  const N = 3;
  const promises = [];
  ENDPOINTS.forEach(endpoint => {
    const axios = getAxios(endpoint);
    API_PATHS.forEach(path => {
      Array(N)
        .fill(0)
        .forEach((x, idx) => {
          promises.push(oneApiCall(axios, endpoint, path, idx));
        });
    });
  });
  await seqPromiseAll(promises);
}

async function testRestApiEndpointPathNtimesPerf(axios, endpoint, path, n) {
  console.log(
    `>>> testRestApiEndpointPathNtimesPerf { endpoint=${endpoint}, path=${path}, n=${n}`
  );
  const st = Date.now();
  try {
    const data = await seqPromiseAll(
      Array(n)
        .fill(0)
        .map(x => axios.get(path).then(res => res.data))
    );
    console.log(
      `>>> testRestApiEndpointPathNtimesPerf } endpoint=${endpoint}, path=${path}, n=${n}`
    );
    return {
      time: Date.now() - st,
      // data: [], // data,
    };
  } catch (e) {
    console.log('>>> error:', e);
    return {
      time: Date.now() - st,
      // include error e can cause JSON.stringify issue
      error: true, // e,
    };
  }
}
async function testAggRestApiEndpointNtimesPerf(axios, endpoint, n) {
  const st = Date.now();
  const payload = {
    field: 'edgeId',
    type: 'datasource',
  };
  try {
    const data = await Promise.all(
      Array(n)
        .fill(0)
        .map(x =>
          axios.post('/v1/common/aggregates', payload).then(res => res.data)
        )
    );
    return {
      time: Date.now() - st,
      // data,
    };
  } catch (e) {
    console.log('>>> error:', e);
    return {
      time: Date.now() - st,
      // include error e can cause JSON.stringify issue
      error: true, // e,
    };
  }
}
async function testRestApiEndpointPathPerf(axios, endpoint, path) {
  const iters = await seqPromiseAll(
    ITERATIONS.map(n =>
      testRestApiEndpointPathNtimesPerf(axios, endpoint, path, n)
    )
  );
  const rs = iters.reduce((acc, cur, i) => {
    acc[ITERATIONS[i]] = cur;
    return acc;
  }, {});
  // console.log(
  //   `testRestApiEndpointPathPerf(endpoint=${endpoint}, path=${path}), rs=`,
  //   JSON.stringify(rs, null, 2)
  // );
  return rs;
}
async function testRestApiEndpointPerf(endpoint) {
  const axios = getAxios(endpoint);
  const ps = await seqPromiseAll(
    API_PATHS.map(path =>
      testRestApiEndpointPathPerf(axios, endpoint, `/v1/${path}`)
    )
  );
  const rs = ps.reduce((acc, cur, i) => {
    acc[API_PATHS[i]] = cur;
    return acc;
  }, {});
  // console.log(
  //   `testRestApiEndpointPerf(endpoint=${endpoint}), rs=`,
  //   JSON.stringify(rs, null, 2)
  // );
  return rs;
}
async function testAggRestApiEndpointPerf(endpoint) {
  const axios = getAxios(endpoint);

  const iters = await Promise.all(
    ITERATIONS.map(n => testAggRestApiEndpointNtimesPerf(axios, endpoint, n))
  );
  const rs = iters.reduce((acc, cur, i) => {
    acc[ITERATIONS[i]] = cur;
    return acc;
  }, {});
  return rs;
}
async function testRestApiPerf() {
  const es = await seqPromiseAll(
    ENDPOINTS.map(endpoint => testRestApiEndpointPerf(endpoint))
  );
  const rs = es.reduce((acc, cur, i) => {
    acc[ENDPOINTS[i]] = cur;
    return acc;
  }, {});
  // console.log(`testRestApiPerf(), rs=`, JSON.stringify(rs, null, 2));
  return rs;
}

async function testAggRestApiPerf() {
  const es = await Promise.all(
    ENDPOINTS.map(endpoint => testAggRestApiEndpointPerf(endpoint))
  );
  const rs = es.reduce((acc, cur, i) => {
    acc[ENDPOINTS[i]] = cur;
    return acc;
  }, {});
  // console.log(`testRestApiPerf(), rs=`, JSON.stringify(rs, null, 2));
  return rs;
}

async function testRestApi() {
  const startTime = Date.now();
  const ps = await Promise.all(
    Array(10)
      .fill(0)
      .map(x => axiosInstance.get(API_PATH).then(res => res.data))
  );
  console.log(
    `got edges in ${Date.now() - startTime}ms`,
    JSON.stringify(ps, null, 2)
  );
}

function testGetEdgeHandleToken() {
  const edgeId = '8ee9b069-8ad0-4313-a6d3-bbdbf6e7f4a9';
  console.log(`token for edge ${edgeId} is ${getEdgeHandleToken(edgeId)}`);
}
async function testCrypto() {
  Array(10)
    .fill(0)
    .forEach(async x => {
      const password = await crypto2.createPassword();
      console.log('password:', password);
    });

  const password = 'seB9wmU0pz/qQ/auR2vMVit0hEE4VnUq';
  const msg =
    'hello world this can be a really long message to encrypt and we do on and on and blah blah blan, lorem pium etc etc';
  const encrypted = await crypto2.encrypt(msg, password);
  const decrypted = await crypto2.decrypt(encrypted, password);
  console.log('encrypted:', encrypted);
  console.log('decrypted:', decrypted);
}

function asciiToBase64(str) {
  return Buffer.from(str).toString('base64');
}
function base64ToAscii(str) {
  return Buffer.from(str, 'base64').toString('ascii');
}

function stringToBuffer(str) {
  return Buffer.from(str, 'utf8');
}
function bufferToString(buf) {
  // return buf.toString('utf8');
  return buf.toString('base64');
}

function kmsGenerateDataKey(kms, params) {
  return new Promise((resolve, reject) => {
    kms.generateDataKey(params, function(err, data) {
      if (err) {
        reject(err);
      } else {
        resolve(data);
      }
    });
  });
}

function kmsDecrypt(kms, params) {
  return new Promise((resolve, reject) => {
    kms.decrypt(params, function(err, data) {
      if (err) {
        reject(err);
      } else {
        resolve(data);
      }
    });
  });
}

async function testKMS() {
  AWS.config.update(<any>{
    region: 'us-west-2',
  });
  const kms = new AWS.KMS({ apiVersion: '2014-11-01' });
  /* The following example generates a 256-bit symmetric data encryption key (data key) in two formats. One is the unencrypted (plainext) data key, and the other is the data key encrypted with the specified customer master key (CMK). */

  const params = {
    KeyId: 'alias/ntnx/cloudmgmt-dev', // The identifier of the CMK to use to encrypt the data key. You can use the key ID or Amazon Resource Name (ARN) of the CMK, or the name or ARN of an alias that refers to the CMK.
    KeySpec: 'AES_256', // Specifies the type of data key to return.
  };
  try {
    const data: any = await kmsGenerateDataKey(kms, params);
    const { CiphertextBlob, Plaintext } = data;
    console.log(data);
    console.log('CiphertextBlob:', bufferToString(CiphertextBlob));
    console.log('Plaintext:', bufferToString(Plaintext));
    const dk1 = base64ToAscii(bufferToString(Plaintext));
    const pt: any = await kmsDecrypt(kms, { CiphertextBlob });
    console.log('decrypt data:', pt);
    console.log('decrypt Plaintext:', bufferToString(pt.Plaintext));
    const dk2 = base64ToAscii(bufferToString(pt.Plaintext));
    console.log('dk1 === dk2? ' + (dk1 === dk2), dk1);
    console.log('dk1.length: ' + dk1.length);
    console.log(
      'password.length: ' + 'seB9wmU0pz/qQ/auR2vMVit0hEE4VnUq'.length
    );

    // next, decrypt data key
  } catch (err) {
    console.log(err, err.stack);
  }
}

async function _testKeyService(keyService: KeyService) {
  const msg =
    'Hello darkness my old friend, I have come to talk with you again!';
  const tenantToken = await keyService.genTenantToken();
  console.log('using tenant token: ', tenantToken);
  const enc = await keyService.tenantEncrypt(msg, tenantToken);
  console.log('encoding: ', enc);
  const dec = await keyService.tenantDecrypt(enc, tenantToken);
  console.log('dec === msg?', dec === msg);

  const payload = {
    tenantId: 'tenant-id-waldot',
    scopes: ['admin', 'foo'],
  };
  const jwtToken = await keyService.jwtSign(payload);
  console.log('signed jwt token is:', jwtToken);
  const jwtDecoded = await keyService.jwtVerify(jwtToken);
  console.log('Got decoded JWT: ', jwtDecoded);
}

async function testKeyService() {
  await _testKeyService(cryptoKeyService);
  const awsKeyService = new AwsKeyService();
  await _testKeyService(awsKeyService);
}

async function testEdgeLogin() {
  const edgeId = uuidv4();
  const { privateKey, publicKey } = await crypto2.createKeyPair();
  console.log('got private key:', privateKey);
  console.log('got public key:', publicKey);

  const signature = await crypto2.sign(edgeId, privateKey);
  console.log('signature:', signature);

  const isSignatureValid = await crypto2.verify(edgeId, publicKey, signature);
  console.log('signature valid? ', isSignatureValid);
}

function toArrayBuffer(buf) {
  var ab = new ArrayBuffer(buf.length);
  var view = new Uint8Array(ab);
  for (var i = 0; i < buf.length; ++i) {
    view[i] = buf[i];
  }
  return ab;
}

function getPublicKeyFromCertificate(certData) {
  const pki = forge.pki;
  const cert = pki.certificateFromPem(certData);
  const pubPem = pki.publicKeyToPem(cert.publicKey);
  const key = new NodeRSA(pubPem);
  return key.exportKey('public');
}

async function testParseCertificate() {
  fs.readFile('./kk.cert', 'utf8', async (err, data) => {
    if (err) throw err;
    console.log(data);
    // const asn1 = asn1js.fromBER(toArrayBuffer(data));
    // const certificate = new pkijs.Certificate({ schema: asn1.result });
    // certificate.
    // console.log('Got certificate:', certificate);
    try {
      // const pki = forge.pki;
      // const cert = pki.certificateFromPem(data);

      // console.log('got cert:', cert);
      // cert.publicKey

      // load public / private key using crypto2
      // sign with private key
      // verify with public key

      // also verify with public key from cert

      const privateKey = await crypto2.readPrivateKey('kk.pem');
      const publicKey = await crypto2.readPublicKey('kk.pub');
      const edgeId = uuidv4();
      const signature = await crypto2.sign(edgeId, privateKey);
      const isSignatureValid = await crypto2.verify(
        edgeId,
        publicKey,
        signature
      );
      console.log('signature valid? ', isSignatureValid);

      const publicKey2 = getPublicKeyFromCertificate(data); //key.exportKey('public');
      const isSignatureValid2 = await crypto2.verify(
        edgeId,
        publicKey2,
        signature
      );
      console.log('signature valid? ', isSignatureValid2);
    } catch (e) {
      console.log('caught error', e);
    }
  });
}

async function main() {
  // let sql = null;
  // if (isSQL()) {
  //   sql = initSequelize();
  // }
  // // await testCreateTenant();
  // // testSha();
  // await testApiPerf();

  // if (sql) {
  //   sql.close();
  // }

  // testRestApi();
  // const rs = await testRestApiPerf();
  // const rs = await testAggRestApiPerf();
  // console.log('testRestApiPerf result:', JSON.stringify(rs, null, 2));

  // await allApiCalls();

  // testGetEdgeHandleToken();

  // await testCrypto();

  // await testKMS();

  // await testKeyService();

  // await testEdgeLogin();

  await testParseCertificate();
}

main();
