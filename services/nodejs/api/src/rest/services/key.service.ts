export interface KeyService {
  genTenantToken(): Promise<string>;

  tenantEncrypt(str: string, token: string): Promise<string>;

  tenantDecrypt(str: string, token: string): Promise<string>;

  jwtSign(payload: any): Promise<string>;

  jwtVerify(token: string): Promise<any>;
}
