import * as uuidv4 from 'uuid/v4';
import {
  randomAttribute,
  randomStringArray,
  randomCount,
  pick,
  arrayEquals,
} from '../common';

export function randomCategory(ctx: any, apiVersion: string) {
  const { tenantId } = ctx;
  const id = uuidv4();
  const name = randomAttribute('name');
  const purpose = randomAttribute('purpose');
  const values = randomStringArray(randomCount(2, 10), 'values').sort();
  return {
    id,
    tenantId,
    name,
    purpose,
    values,
  };
}

export function randomCategoryUpdate(ctx: any, apiVersion: string, entity) {
  const updated = randomCategory(ctx, apiVersion);
  const { id } = entity;
  const values = [...entity.values, ...updated.values].sort();
  return { ...updated, values, id };
}

export function purifyCategory(category: any, apiVersion: string) {
  const { id, tenantId, name, purpose, values } = category;
  values.sort();
  return {
    id,
    tenantId,
    name,
    purpose,
    values,
  };
}

export function randomCategoryInfo(ctx: any) {
  const { categories } = ctx;
  const { id, values } = pick(categories);
  const value = pick(values);
  return {
    id,
    value,
  };
}

export function dedupeCategoryInfos(catInfoList: any[]): any[] {
  const m: any = {};
  return catInfoList.filter(catInfo => {
    const key = `${catInfo.id}:${catInfo.value}`;
    if (m[key]) {
      return false;
    } else {
      m[key] = true;
      return true;
    }
  });
}
