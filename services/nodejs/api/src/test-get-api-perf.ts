import { seqPromiseAll, getAxios, logTime } from './test-common';

async function oneApiCall(axios, endpoint, path, idx) {
  const st = Date.now();
  let error = false;
  try {
    const data = await axios.get(path).then(res => res.data);
  } catch (e) {
    console.log('>>> error:', (e.response && e.response.data) || e);
    error = true;
  }
  logTime(
    [
      { key: 'endpoint', val: endpoint },
      { key: 'path', val: path },
      { key: 'idx', val: idx },
    ],
    {
      time: Date.now() - st,
      error,
    }
  );
}

function getPath(api) {
  if (api.match(/edge/i)) {
    return '/v1/edges';
  } else if (api.match(/category/i)) {
    return '/v1/categories';
  } else if (api.match(/datasource/i)) {
    return '/v1/datasources';
  } else if (api.match(/datastream/i)) {
    return 'v1/datastreams';
  } else if (api.match(/script/i)) {
    return '/v1/scripts';
  } else if (api.match(/sensor/i)) {
    return '/v1/sensors';
  }
  throw Error(`Unknown api: ${api}`);
}

const USAGE = `\nUsage: node test-get-api-perf.js <endpoint> <tenant id> <api> <iteration>\n`;

// example endpoint: http://localhost:3000, https://sherlockntnx.com, https://ntnxsherlock.com
// example tenant id: tenant-id-waldot
// example api: edge, category, datasource, datastream, script, sensor
// iteration is a positive integer
async function main() {
  if (process.argv.length < 6) {
    console.log(USAGE);
    process.exit(1);
  }
  const endpoint = process.argv[2];
  const tenantId = process.argv[3];
  const path = getPath(process.argv[4]);
  const iter = process.argv[5];
  const axios = getAxios(endpoint, tenantId);
  const promises = Array(parseInt(iter))
    .fill(0)
    .map((x, idx) => oneApiCall(axios, endpoint, path, idx));

  // await seqPromiseAll(promises);
  await Promise.all(promises);
}

main();
