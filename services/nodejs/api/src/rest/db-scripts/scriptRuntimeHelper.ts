import { DataTypes } from 'sequelize';
import { SCRIPT_RUNTIMES } from './dataDB';

export async function createBuiltinScriptRuntimes(sql, tenantId) {
  const ScriptRuntime = sql.define(
    'ScriptRuntime',
    {
      id: { type: DataTypes.STRING, primaryKey: true },
      name: DataTypes.STRING,
      description: DataTypes.STRING,
      tenant_id: DataTypes.STRING,
      version: DataTypes.INTEGER,
      language: DataTypes.STRING,
      dockerfile: DataTypes.STRING,
      docker_repo_uri: DataTypes.STRING,
      docker_profile_id: DataTypes.STRING,
      builtin: DataTypes.BOOLEAN,
      created_at: DataTypes.TIME,
      updated_at: DataTypes.TIME,
    },
    {
      freezeTableName: true,
      // define the table's name
      tableName: 'script_runtime_model',
      timestamps: true,
      createdAt: 'created_at',
      updatedAt: 'updated_at',
    }
  );
  return Promise.all(
    SCRIPT_RUNTIMES.map(scriptRuntime => {
      let sr = { ...scriptRuntime };
      sr.id = `${tenantId}_${scriptRuntime.id}`;
      sr.tenant_id = tenantId;
      return ScriptRuntime.upsert(sr);
    })
  );
}
