/**
 * Simple global (singleton) in-memory registry service.
 * To facilitate global register and lookup.
 */
export interface RegistryService {
  register(name: string, obj: any): void;
  unregister(obj: any): string;
  get(name: string): any;
}

class RegisterServiceImpl implements RegistryService {
  private reg: any = {};
  public register(name: string, obj: any): void {
    this.reg[name] = obj;
  }
  public unregister(obj: any): string {
    const keys = Object.keys(this.reg);
    const key = keys.find(k => this.reg[k] === obj);
    if (key) {
      this.reg[key] = null;
      return key;
    }
    return null;
  }
  public get(name: string): any {
    return this.reg[name];
  }
}

// key used by websocket server to register itself
export const REG_KEY_SOCKET_IO = 'socket.io';

// key used by websocket client to register itself
export const REG_KEY_SOCKET_IO_CLIENT = 'socket.io.client';

export const REG_KEY_KAFKA_SERVICE = 'kafka.service';

export const registryService: RegistryService = new RegisterServiceImpl();
