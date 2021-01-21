import AxiosLib from 'axios';

export function getAxios(endpoint, tenantId) {
  return AxiosLib.create({
    baseURL: endpoint,
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: 60000,
  });
}

export function seqPromiseAll(promises, acc = [], idx = 0) {
  console.log('seqPromiseAll { idx=', idx);
  if (!promises || !promises.length || idx >= promises.length) {
    return acc;
  }
  const promise = promises[idx];
  return promise.then(r => {
    acc.push(r);
    return seqPromiseAll(promises, acc, idx + 1);
  });
}

export function logTime(arr, tobj) {
  const s = arr.map(a => `${a.key}=${a.val}`).join(', ');
  // TODO
  const err = tobj.error ? 'error=true' : '';
  console.log(`logTime: ${s}, time=${tobj.time}, ${err}`);
}
