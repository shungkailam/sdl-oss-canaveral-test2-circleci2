import { DocType } from '../model/baseModel';
import { User } from '../model/user';
import { getDBService } from '../db-configurator/dbConfigurator';
import platformService from '../services/platform.service';

export async function getAllUsers(tenantId: string): Promise<User[]> {
  return getDBService().getAllDocuments<User>(tenantId, DocType.User);
}

export function getUserByEmail(email: string): Promise<User> {
  return getDBService().getUserByEmail(email);
}

export async function createAdminToken(user: User) {
  const { tenantId, email, id, name } = user;
  const payload = {
    tenantId,
    email,
    id,
    name,
    // TODO FIXME - derive scopes from user roles
    specialRole: 'admin',
    roles: [],
    scopes: [],
  };
  const token = await platformService.getKeyService().jwtSign(payload);
  return { token };
}
