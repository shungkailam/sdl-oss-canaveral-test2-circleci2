import { Injectable } from '@angular/core';

@Injectable()
export class RegistryService {
  private reg: any = {};

  public register(name: string, obj: any): void {
    this.reg[name] = obj;
  }
  public get(name: string): any {
    return this.reg[name];
  }
}

export const REG_KEY_TENANT_ID = 'tenant.id';
