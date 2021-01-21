import {
  InitializerServiceImpl,
  InitializerService,
} from '../rest/services/initializer.service';

class CloudInitializerServiceImpl extends InitializerServiceImpl {}

export const cloudInitializer: InitializerService = new CloudInitializerServiceImpl();
