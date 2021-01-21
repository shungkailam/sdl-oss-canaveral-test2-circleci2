import { KeyService } from '../key.service';
import * as crypto2 from 'crypto2';
import { encrypt, decrypt } from '../../../rest/util/cryptoUtil';
import * as jwt from 'jsonwebtoken';

class CryptoKeyService implements KeyService {
  private sherlockPassword =
    process.env.CRYPTO_PASSWORD || 'seB9wmU0pz/qQ/auR2vMVit0hEE4VnUq';

  private jwtSecret =
    process.env.JWT_SECRET || 'jktYsiqtJHaWe8ow3dhCogrYAJwQiQ6D';

  public async genTenantToken(): Promise<string> {
    const p = await crypto2.createPassword();
    return encrypt(p, this.sherlockPassword);
  }

  public async tenantEncrypt(str: string, token: string): Promise<string> {
    const tenantPassword = await decrypt(token, this.sherlockPassword);
    return encrypt(str, tenantPassword);
  }

  public async tenantDecrypt(str: string, token: string): Promise<string> {
    const tenantPassword = await decrypt(token, this.sherlockPassword);
    return decrypt(str, tenantPassword);
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

const keyService: KeyService = new CryptoKeyService();

export default keyService;
