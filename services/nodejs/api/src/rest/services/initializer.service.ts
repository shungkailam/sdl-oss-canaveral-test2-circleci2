import { registryService } from './registry.service';
import { setupGlobalIndexMaybe } from '../db-scripts/commonDB';
import { CreateDocumentResponse } from '../model/baseModel';

interface RegistryInitializer {
  key: string;
  value: any;
}

export interface InitializerService {
  initDB(esClient: Object, initialConfig: Object): Promise<void>;
  initRegistryService(obj: RegistryInitializer[]): void;
  postInit(
    esClient: Object,
    initialConfig: Object
  ): Promise<CreateDocumentResponse>;
}

const WAIT_ES = 2000;

export class InitializerServiceImpl implements InitializerService {
  public async initDB(esClient: Object, initialConfig: Object) {
    await this.initIndex(esClient);
    await this.postInit(esClient, initialConfig);
    return this.waitForElasticSearch();
  }

  public async postInit(esClient, initialConfig) {
    // Implement otherwise NO-OP.
    // TODO: Can we return a generic promise?
    return new Promise<CreateDocumentResponse>((resolve, reject) => {
      resolve();
    });
  }

  private waitForElasticSearch(): Promise<void> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        // Do nothing just wait.
        resolve();
      }, WAIT_ES);
    });
  }

  private async initIndex(esClient) {
    await setupGlobalIndexMaybe();
  }

  public initRegistryService(obj: RegistryInitializer[]): void {
    obj.forEach(kv => {
      if (registryService.get(kv.key)) {
        console.log('Key already present, overriding value', kv.key);
      }

      registryService.register(kv.key, kv.value);
    });
  }
}
