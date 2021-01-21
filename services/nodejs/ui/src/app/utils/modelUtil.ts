import { DataStream, DataSource, CategoryInfo } from '../model/index';

export function datastreamContainsDatasource(
  stream: DataStream,
  src: DataSource
): boolean {
  if (stream.originId) {
    return false;
  }
  return datasourceMatchOriginSelectors(
    src,
    stream.originSelectors,
    stream.dataType
  );
}

export function datasourceMatchOriginSelectors(
  src,
  originSelectors,
  dataType: string
): boolean {
  // for simplicity, treat data source field selector scope as ALL
  // Note: for multiple selectors for the same category, the semantics is 'OR'
  // while across different categories the semantics is 'AND'

  // map: category id -> CategoryInfo[]
  const selectorGroupMap: any = {};
  originSelectors = originSelectors || [];
  originSelectors.forEach(sel => {
    let list = selectorGroupMap[sel.id];
    if (!list) {
      list = [sel];
      selectorGroupMap[sel.id] = list;
    } else {
      list.push(sel);
    }
  });
  const selectorGroups = Object.keys(selectorGroupMap).map(
    k => selectorGroupMap[k]
  );
  let match = selectorGroups.every(sels => {
    const sid = sels[0].id;
    const vals = sels.map(sel => sel.value);
    return src.selectors.some(ss => {
      return ss.id === sid && vals.indexOf(ss.value) !== -1;
    });
  });
  return match;
}

export function getDatasourceSensorsCount(dataSource: DataSource): number {
  const fdtMap = {};
  dataSource.fields.forEach(f => {
    fdtMap[f.name] = true;
  });
  return Object.keys(fdtMap).length;
}

export function getDatasourceMatchingSensorsCount(
  dataSource: DataSource,
  originSelectors: CategoryInfo[],
  dataType: string
): number {
  const allfdMap = {};
  const fdtMap = {};
  dataSource.fields.forEach(f => {
    allfdMap[f.name] = true;
    fdtMap[f.name] = true;
  });
  if (Object.keys(fdtMap).length === 0) {
    return 0;
  }

  // map: category id -> CategoryInfo[]
  const selectorGroupMap: any = {};
  originSelectors = originSelectors || [];
  originSelectors.forEach(sel => {
    let list = selectorGroupMap[sel.id];
    if (!list) {
      list = [sel];
      selectorGroupMap[sel.id] = list;
    } else {
      list.push(sel);
    }
  });
  const selectorGroups = Object.keys(selectorGroupMap).map(
    k => selectorGroupMap[k]
  );
  const sgfdMap = selectorGroups.map(sg => ({}));
  selectorGroups.forEach((sg, i) => {
    const sgId = sg[0].id;
    for (let j = 0; j < dataSource.selectors.length; j++) {
      const sel = dataSource.selectors[j];
      if (sel.id === sgId && sg.some(s => s.value === sel.value)) {
        if (sel.scope.length === 1 && sel.scope[0] === '__ALL__') {
          sgfdMap[i] = allfdMap;
          break;
        } else {
          const sgfdMapi = sgfdMap[i];
          sel.scope.forEach(f => (sgfdMapi[f] = true));
        }
      }
    }
  });
  let count = 0;
  Object.keys(allfdMap).forEach(f => {
    if (fdtMap[f] && sgfdMap.every(m => m[f])) {
      count++;
    }
  });
  return count;
}
