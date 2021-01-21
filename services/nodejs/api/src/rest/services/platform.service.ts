import { KeyService } from './key.service';
// import cryptoKeyService from './impl/key.service.crypto';
import { AwsKeyService } from './impl/key.service.aws';
import { GcmKeyService } from './impl/key.service.gcm';

export interface PlatformService {
  getKeyService(): KeyService;
}

class PlatformServiceImpl implements PlatformService {
  private keyService: KeyService = null;

  public getKeyService(): KeyService {
    if (this.keyService === null) {
      if (process.env.USE_KMS === 'false') {
        this.keyService = new GcmKeyService();
      } else {
        this.keyService = new AwsKeyService();
      }
    }
    return this.keyService;
  }
}

const platformService: PlatformService = new PlatformServiceImpl();

export default platformService;
