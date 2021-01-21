import { Sequelize } from 'sequelize-typescript';
import { SCRIPT_RUNTIMES } from './dataDB';
import { createDocNew, TenantBuilder } from './tenantBuilder';
import { getVersion } from './createDataHelper';
import { DocType, Category } from '../model';

function getDataTypeCategoryId(tenantId: string): string {
  return `${tenantId}-cat-data-type`;
}

export async function createDataTypeCategory(sql, tenantId) {
  var tb = new TenantBuilder(null, sql); // need this to initialize sql
  const id = getDataTypeCategoryId(tenantId);
  const cat: Category = {
    tenantId,
    id,
    version: getVersion(),
    name: 'Data Type',
    purpose: 'To specify data type for each field in a data source.',
    values: [
      'Custom',
      'Humidity',
      'Image',
      'Light',
      'Motion',
      'Pressure',
      'Processed',
      'Proximity',
      'Temperature',
    ],
  };
  return createDocNew(tenantId, DocType.Category, cat);
}

export function addDataTypeCategorySelectors(tenantId, fields, selectors) {
  const id = getDataTypeCategoryId(tenantId);
  fields.forEach(f => {
    selectors.push({
      id,
      value: f.fieldType,
      scope: [f.name],
    });
  });
}
