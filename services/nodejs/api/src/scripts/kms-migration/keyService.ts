import * as AWS from 'aws-sdk';
import { encrypt, decrypt } from '../../rest/util/cryptoUtil';
import {
  kmsGenerateDataKey,
  kmsDecrypt,
  AWS_REGION,
} from '../../rest/services/impl/key.service.aws';

const BASE64 = 'base64';

export class AwsKmsKeyService {
  private kms: any = null;

  constructor() {
    AWS.config.update(<any>{
      region: AWS_REGION,
    });
    this.kms = new AWS.KMS({ apiVersion: '2014-11-01' });
  }

  public async genTenantToken(keyId: string): Promise<string> {
    return this.genDataKey(keyId);
  }

  public async genJWTSecret(keyId: string): Promise<string> {
    return this.genDataKey(keyId);
  }

  public async genDataKey(keyId: string): Promise<string> {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const params = {
          KeyId: keyId,
          KeySpec: 'AES_256',
        };
        const data: any = await kmsGenerateDataKey(this.kms, params);
        const { CiphertextBlob } = data;
        resolve(CiphertextBlob.toString(BASE64));
      } catch (err) {
        console.log(`genTenantToken: failed, keyId=${keyId}`);
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
        console.log(`kmsDecryptToken: failed, token=${token}`);
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
        console.log(`tenantEncrypt: failed, s=${str}, token=${token}`);
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
        console.log(`tenantDecrypt: failed, s=${str}, token=${token}`);
        reject(err);
      }
    });
  }
}
