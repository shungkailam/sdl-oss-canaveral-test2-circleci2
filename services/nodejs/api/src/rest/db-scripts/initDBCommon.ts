import { DataTypes } from 'sequelize';
import { initSequelize } from '../sql-api/baseApi';
import { TENANT_ID, TENANT_ID_2, getUser, SCRIPT_RUNTIMES } from './dataDB';
import { DocType, DocTypes } from '../model/baseModel';
import { getDBService, isSQL } from '../db-configurator/dbConfigurator';
import platformService from '../services/platform.service';
import { createBuiltinScriptRuntimes } from './scriptRuntimeHelper';
import { createDataTypeCategory } from './categoryHelper';
import { createDefaultProject } from '../../scripts/common';

var sql = null;

// main function
// declare as async so we can use ES7 async/await
async function main() {
  if (isSQL()) {
    sql = initSequelize();
  }
  try {
    console.log('Creating new tenant records');
    await createTenantUser(TENANT_ID, 'waldot');
    await createTenantUser(TENANT_ID_2, 'Rocket Blue');
  } finally {
    if (isSQL()) {
      await sql.close();
    }
  }
}

async function createTenantUser(tenantId, tenantName): Promise<void> {
  return new Promise<void>(async (resolve, reject) => {
    try {
      const tenantToken = await platformService
        .getKeyService()
        .genTenantToken();
      const date = new Date();
      // create tenant
      const doc = {
        id: tenantId,
        version: 0,
        name: tenantName,
        description: 'My company',
        token: tenantToken,
        created_at: date,
        updated_at: date,
      };
      const Tenant = sql.define(
        'Tenant',
        {
          id: { type: DataTypes.STRING, primaryKey: true },
          name: DataTypes.STRING,
          description: DataTypes.STRING,
          token: DataTypes.STRING,
          version: DataTypes.INTEGER,
          created_at: DataTypes.TIME,
          updated_at: DataTypes.TIME,
        },
        {
          freezeTableName: true,
          // define the table's name
          tableName: 'tenant_model',
          timestamps: true,
          createdAt: 'created_at',
          updatedAt: 'updated_at',
        }
      );

      try {
        const t = await Tenant.findOne({
          where: {
            id: doc.id,
          },
          raw: true,
        });

        if (t) {
          console.log('tenant already exist, skip create', t);
          resolve(null);
          return;
        }
      } catch (e) {
        console.log('find tenant error', e);
      }

      // create tenant
      await Tenant.upsert(doc);

      // create builtin script runtimes
      await createBuiltinScriptRuntimes(sql, tenantId);

      // create data type category
      await createDataTypeCategory(sql, tenantId);

      const user = getUser(tenantId);
      if (user) {
        const doc = {
          id: user.id,
          tenant_id: tenantId,
          email: user.email,
          name: user.name,
          password: user.password,
          version: 0,
          created_at: date,
          updated_at: date,
        };
        const User = sql.define(
          'User',
          {
            id: { type: DataTypes.STRING, primaryKey: true },
            tenant_id: DataTypes.STRING,
            email: DataTypes.STRING,
            name: DataTypes.STRING,
            password: DataTypes.STRING,
            version: DataTypes.INTEGER,
            created_at: DataTypes.TIME,
            updated_at: DataTypes.TIME,
          },
          {
            freezeTableName: true,
            // define the table's name
            tableName: 'user_model',
            timestamps: true,
            createdAt: 'created_at',
            updatedAt: 'updated_at',
          }
        );
        await User.upsert(doc);
      }

      await createDefaultProject(sql, tenantId, false);

      resolve(null);
    } catch (err) {
      reject(err);
    }
  });
}

main();
