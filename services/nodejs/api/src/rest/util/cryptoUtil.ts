import * as sha256 from 'sha256';
import * as crypto2 from 'crypto2';
import * as crypto from 'crypto';
import * as jwt from 'jsonwebtoken';
import * as forge from 'node-forge';
import * as NodeRSA from 'node-rsa';
import { User } from '../model/user';

const salt = sha256('sherlock');

// we use this to encode user password saved in DB
export function getSha256(str: string): string {
  return sha256(str + salt);
}

// token required for edge to retrieve private key
// note, for security reason, the API endpoint
// for an edge to retrieve its private key only works
// on the first invocation, not any subsequent invocations
export function getEdgeHandleToken(edgeId: string): string {
  return getSha256(edgeId);
}

// encrypt using the AES 256 CBC encryption algorithm
export function encrypt(str: string, password: string): Promise<string> {
  const ivb = crypto.randomBytes(12);
  const cipher = crypto.createCipheriv(
    'aes-256-gcm',
    Buffer.from(password, 'base64'),
    ivb
  );
  let enc = cipher.update(str, 'utf8', 'hex');
  enc += cipher.final('hex');
  const buf = Buffer.concat([
    ivb,
    Buffer.from(enc, 'hex'),
    cipher.getAuthTag(),
  ]);
  return Promise.resolve(buf.toString('hex'));
}
// decrypt using the AES 256 CBC encryption algorithm
export function decrypt(str: string, password: string): Promise<string> {
  const decoded = Buffer.from(str, 'hex');
  const nonce = decoded.slice(0, 12);
  const ciphertext = decoded.slice(12, decoded.length - 16);
  const tag = decoded.slice(decoded.length - 16);
  const decipher = crypto.createDecipheriv(
    'aes-256-gcm',
    Buffer.from(password, 'base64'),
    nonce
  );
  decipher.setAuthTag(tag);
  let plaintext = decipher.update(ciphertext, null, 'utf8');
  plaintext += decipher.final('utf8');
  return Promise.resolve(plaintext);
}

export function getPublicKeyFromCertificate(certData) {
  const pki = forge.pki;
  const cert = pki.certificateFromPem(certData);
  const pubPem = pki.publicKeyToPem(cert.publicKey);
  const key = new NodeRSA(pubPem);
  return key.exportKey('public');
}

export function encryptUserPassword(user: User) {
  user.password = encryptPassword(user.password);
}

export function encryptPassword(password: string): string {
  return getSha256(password);
}
