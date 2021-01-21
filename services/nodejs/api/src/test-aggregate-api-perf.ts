import { seqPromiseAll, getAxios, logTime } from './test-common';

async function oneApiCall(axios, endpoint, type, field, idx) {
  const st = Date.now();
  let error = false;
  try {
    const payload = {
      field,
      type,
    };
    const data = await axios
      .post('/v1/common/aggregates', payload)
      .then(res => res.data);
  } catch (e) {
    console.log('>>> error:', (e.response && e.response.data) || e);
    error = true;
  }

  logTime(
    [
      { key: 'endpoint', val: endpoint },
      { key: 'type', val: type },
      { key: 'field', val: field },
      { key: 'idx', val: idx },
    ],
    {
      time: Date.now() - st,
      error,
    }
  );
}

const USAGE = `\nUsage: node test-aggregate-api-perf.js <endpoint> <tenant id> <type> <field> <iteration>\n`;

// example endpoint: http://localhost:3000, https://sherlockntnx.com, https://ntnxsherlock.com
// example tenant id: tenant-id-waldot
// example api: edge, category, datasource, datastream, script, sensor
// iteration is a positive integer
async function main() {
  if (process.argv.length < 7) {
    console.log(USAGE);
    process.exit(1);
  }
  const endpoint = process.argv[2];
  const tenantId = process.argv[3];
  const type = process.argv[4];
  const field = process.argv[5];
  const iter = process.argv[6];
  const axios = getAxios(endpoint, tenantId);
  const promises = Array(parseInt(iter))
    .fill(0)
    .map((x, idx) => oneApiCall(axios, endpoint, type, field, idx));

  // await seqPromiseAll(promises);
  await Promise.all(promises);
}

main();
