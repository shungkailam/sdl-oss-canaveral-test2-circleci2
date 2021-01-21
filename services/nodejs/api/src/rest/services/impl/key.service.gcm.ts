import { KeyService } from '../key.service';
import * as jwt from 'jsonwebtoken';
import * as crypto from 'crypto';
const nonceSize = 12;
const gcmTagSize = 16;
const algorithm = 'aes-256-gcm';

// see: https://jg.gg/2018/01/22/communicating-via-aes-256-gcm-between-nodejs-and-golang/
function decrypt(hexStr: string, aesKey: Buffer): string {
  const decoded = Buffer.from(hexStr, 'hex');
  const nonce = decoded.slice(0, nonceSize);
  const cipherText = decoded.slice(nonceSize, decoded.length - gcmTagSize);
  const tag = decoded.slice(decoded.length - gcmTagSize);
  const decipher = crypto.createDecipheriv(algorithm, aesKey, nonce);
  decipher.setAuthTag(tag);
  let plaintext = decipher.update(cipherText, null, 'utf8');
  plaintext += decipher.final('utf8');
  return plaintext;
}

function encrypt(text: string, aesKey: Buffer): string {
  const iv = crypto.randomBytes(nonceSize);
  const cipher = crypto.createCipheriv(algorithm, aesKey, iv);
  let encrypted = cipher.update(text, 'utf8', 'hex');
  encrypted += cipher.final('hex');
  const tag = cipher.getAuthTag();
  return `${iv.toString('hex')}${encrypted}${tag.toString('hex')}`;
}

export class GcmKeyService implements KeyService {
  private jwtSecret: Buffer;
  private masterKey: Buffer;

  constructor() {
    this.masterKey = Buffer.from(process.env.AWS_KMS_KEY, 'base64');
    this.jwtSecret = this.decryptDataKey(process.env.JWT_SECRET);
  }

  private decryptDataKey(b64DataKey: string): Buffer {
    const t = Buffer.from(b64DataKey, 'base64').toString('hex');
    const s = decrypt(t, this.masterKey);
    return Buffer.from(s, 'hex');
  }

  public async genTenantToken(): Promise<string> {
    const baDataKey = crypto.randomBytes(32);
    const hexDataKey = baDataKey.toString('hex');
    const encHexDataKey = encrypt(hexDataKey, this.masterKey);
    return Buffer.from(encHexDataKey, 'hex').toString('base64');
  }

  public async tenantEncrypt(str: string, token: string): Promise<string> {
    const t = this.decryptDataKey(token);
    return encrypt(str, t);
  }

  public async tenantDecrypt(str: string, token: string): Promise<string> {
    const t = this.decryptDataKey(token);
    return decrypt(str, t);
  }

  public async jwtSign(payload: any): Promise<string> {
    return new Promise<any>(async (resolve, reject) => {
      jwt.sign(
        payload,
        this.jwtSecret,
        {
          expiresIn: 60 * 60 * 24, // 1 day
        },
        function(err, token) {
          if (err) {
            reject(err);
          } else {
            resolve(token);
          }
        }
      );
    });
  }

  public async jwtVerify(token: string): Promise<any> {
    return new Promise<any>(async (resolve, reject) => {
      jwt.verify(token, this.jwtSecret, function(err, decoded) {
        if (err) {
          reject(err);
        } else {
          resolve(decoded);
        }
      });
    });
  }
}
