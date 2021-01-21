import { expect } from 'chai';
import 'mocha';
import * as sha256 from 'sha256';
import {
  getSha256,
  getEdgeHandleToken,
  encrypt,
  decrypt,
  getPublicKeyFromCertificate,
} from './cryptoUtil';
import * as crypto2 from 'crypto2';
import * as fs from 'fs';

function readFile(path: string): Promise<string> {
  return new Promise((resolve, reject) => {
    fs.readFile(path, 'utf8', (err, data) => {
      if (err) {
        reject(err);
      } else {
        resolve(data);
      }
    });
  });
}

describe('getSha256', () => {
  it('should return correct format', () => {
    const result = sha256('apex');
    expect(result).to.equal(
      '846be5fc541c5039f7972c6f0a054a3460d3f4b3ee3541991baee4027465abdb'
    );

    const saltyResult = getSha256('apex');
    expect(saltyResult).to.equal(
      '86454ea5bb9fb8dfcc7c035b96f19dc9032944de02490fde30d54527c51f72a6'
    );
  });
});

describe('getEdgeHandleToken', () => {
  it('should return correct format', () => {
    const edgeId = 'c8d4ac78-be72-4369-b162-2f634ac97816';
    const result = getEdgeHandleToken(edgeId);
    expect(result).to.equal(
      '4046d011a88e3772346f323620e6b10ebfb159790dc046bef7c7f01152f4e288'
    );
  });
});

describe('encrypt/decript', () => {
  it('should return correct format', async () => {
    const password = '3dT/Jqbt7V6Y9zM+yC2Ytg2Rb3PBGVXM';
    console.log('password: ', password);
    const msg = 'Hello darkness my old friend...';
    const result = await encrypt(msg, password);
    const msg2 = await decrypt(result, password);
    expect(msg2).to.equal(msg);
    expect(result).to.equal(
      '22bb5a3903cc65646ec4f8cc746ae2a50f25195f4f36773960e198d3bd04d350'
    );
  });
});

describe('getPublicKeyFromCertificate', () => {
  it('should return correct public key', async () => {
    const edgeId = 'c8d4ac78-be72-4369-b162-2f634ac97816';
    const expectedSignature =
      '9cd099659e66bd4a43671191fb5881947323aa61dee5af2fa4d730a56edff1968015b48b04da696dc6a3422d392ce9a6b56b2d4f447736c6b06417659b6cc3b18af4d84d2d9f9ca5f154ba7c307a6bec0f12dceedc21c9ef2aae03124395e2c249eb7791d352abe11e6772a0cde878a4767d62234e60a4ac311a936a5ced3935394378092a2092617259684c625cd8fc03978c2c53967d75cc341f8056487c725c2b091a0e7ebd2b02e257cfc1c82e5018a999289c8688660465188758b9aa28ad73faf0e183388a238ef2330db9fe4c02f983ece44b7a993b15e5836c82a607d3ffc8991df1a24578d42f020092cf0cd1b21ba69820e3efa978427f506ba730';
    const data = await readFile('./test-data/key/foo.cert');

    const privateKey = await crypto2.readPrivateKey('./test-data/key/foo.priv');
    const publicKey = await crypto2.readPublicKey('./test-data/key/foo.pub');
    const signature = await crypto2.sign(edgeId, privateKey);
    expect(signature).to.equal(expectedSignature);
    const isSignatureValid = await crypto2.verify(edgeId, publicKey, signature);
    expect(isSignatureValid).to.equal(true);

    const publicKey2 = getPublicKeyFromCertificate(data); //key.exportKey('public');
    const isSignatureValid2 = await crypto2.verify(
      edgeId,
      publicKey2,
      signature
    );
    expect(isSignatureValid2).to.equal(true);
  });
});
