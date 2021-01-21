import { KeyService } from '../key.service';
import * as AWS from 'aws-sdk';
import { encrypt, decrypt } from '../../../rest/util/cryptoUtil';
import * as jwt from 'jsonwebtoken';

export const AWS_REGION = process.env.AWS_REGION || 'us-west-2';
const CLOUDMGMT_KEY = process.env.AWS_KMS_KEY || 'alias/ntnx/cloudmgmt-dev';
const JWT_SECRET = process.env.JWT_SECRET;
// cache decrypted jwt secret in memory because doing kms decode for every call is way too expensive (+100ms per call)
let gJwtSecret = null;

export function kmsGenerateDataKey(kms, params) {
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

export function kmsDecrypt(kms, params) {
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

const BASE64 = 'base64';

export class AwsKeyService implements KeyService {
  private kms: any = null;

  constructor() {
    AWS.config.update(<any>{
      region: AWS_REGION,
    });
    this.kms = new AWS.KMS({ apiVersion: '2014-11-01' });
  }

  public async genTenantToken(): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const params = {
          KeyId: CLOUDMGMT_KEY,
          KeySpec: 'AES_256',
        };
        const data: any = await kmsGenerateDataKey(this.kms, params);
        const { CiphertextBlob } = data;
        resolve(CiphertextBlob.toString(BASE64));
      } catch (err) {
        reject(err);
      }
    });
  }

  private async kmsDecryptToken(token: string): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const CiphertextBlob = Buffer.from(token, BASE64);
        const pt: any = await kmsDecrypt(this.kms, { CiphertextBlob });
        const tenantPassword = pt.Plaintext.toString(BASE64);
        resolve(tenantPassword);
      } catch (err) {
        reject(err);
      }
    });
  }

  private async kmsDecryptToken2(token: string): Promise<any> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const CiphertextBlob = Buffer.from(token, BASE64);
        const pt: any = await kmsDecrypt(this.kms, { CiphertextBlob });
        resolve(pt.Plaintext);
      } catch (err) {
        reject(err);
      }
    });
  }

  public async tenantEncrypt(str: string, token: string): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const tenantPassword = await this.kmsDecryptToken(token);
        resolve(encrypt(str, tenantPassword));
      } catch (err) {
        reject(err);
      }
    });
  }

  public async tenantDecrypt(str: string, token: string): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const tenantPassword = await this.kmsDecryptToken(token);
        resolve(decrypt(str, tenantPassword));
      } catch (err) {
        reject(err);
      }
    });
  }

  public async jwtSign(payload: any): Promise<string> {
    return new Promise<any>(async (resolve, reject) => {
      try {
        let jwtSecret = gJwtSecret;
        if (!jwtSecret) {
          gJwtSecret = jwtSecret = await this.kmsDecryptToken2(JWT_SECRET);
        }
        jwt.sign(
          payload,
          jwtSecret,
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
      } catch (e) {
        reject(e);
      }
    });
  }

  public async jwtVerify(token: string): Promise<any> {
    return new Promise<any>(async (resolve, reject) => {
      try {
        let jwtSecret = gJwtSecret;
        if (!jwtSecret) {
          gJwtSecret = jwtSecret = await this.kmsDecryptToken2(JWT_SECRET);
        }
        jwt.verify(token, jwtSecret, function(err, decoded) {
          if (err) {
            reject(err);
          } else {
            resolve(decoded);
          }
        });
      } catch (e) {
        reject(e);
      }
    });
  }
}
