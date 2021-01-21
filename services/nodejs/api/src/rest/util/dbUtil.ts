import { DataType } from 'sequelize-typescript';

const SQL_DIALECT = process.env.SQL_DIALECT || 'postgres';
const isPostgresDB = SQL_DIALECT === 'postgres';

export function isPostgres(): boolean {
  return isPostgresDB;
}

/**
 * Get JSON type to use.
 * Postgres requires JSONB, while mysql requires JSON.
 */
export function getJsonType() {
  return isPostgres() ? DataType.JSONB : DataType.JSON;
}
