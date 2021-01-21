import * as uuidv4 from 'uuid/v4';
import * as equal from 'fast-deep-equal';
import { getSha256 } from '../rest/util/cryptoUtil';
import { purifyCategory } from './entities/category';
import { purifyEdge } from './entities/edge';
import { purifyDataSource } from './entities/dataSource';
import { purifyCloudCreds } from './entities/cloudCreds';
import { purifyDockerProfile } from './entities/dockerProfile';
import { purifyUser } from './entities/user';
import { purifyProject } from './entities/project';
import { purifyScriptRuntime } from './entities/scriptRuntime';
import { purifyScript } from './entities/script';
import { purifyApplication } from './entities/application';
import { purifyDataStream } from './entities/dataStream';

//
export interface EntityCreation {
  ctxKey: string;
  entity: string;
}
export const entityCreationList: EntityCreation[] = [
  {
    ctxKey: 'categories',
    entity: 'categories',
  },
  {
    ctxKey: 'edges',
    entity: 'edges',
  },
  {
    ctxKey: 'dataSources',
    entity: 'datasources',
  },
  {
    ctxKey: 'cloudCredss',
    entity: 'cloudcreds',
  },
  {
    ctxKey: 'dockerProfiles',
    entity: 'dockerprofiles',
  },
  {
    ctxKey: 'users',
    entity: 'users',
  },
  {
    ctxKey: 'projects',
    entity: 'projects',
  },
  {
    ctxKey: 'scriptRuntimes',
    entity: 'scriptruntimes',
  },
  {
    ctxKey: 'scripts',
    entity: 'scripts',
  },
  {
    ctxKey: 'applications',
    entity: 'application',
  },
  {
    ctxKey: 'dataStreams',
    entity: 'datastreams',
  },
];

export interface EntityVerification {
  entity: string;
  purifyFn: any;
  ctxKey: string;
}

export const entityVerificationList: EntityVerification[] = [
  {
    entity: 'categories',
    purifyFn: purifyCategory,
    ctxKey: 'categories',
  },
  {
    entity: 'edges',
    purifyFn: purifyEdge,
    ctxKey: 'edges',
  },
  {
    entity: 'datasources',
    purifyFn: purifyDataSource,
    ctxKey: 'dataSources',
  },
  {
    entity: 'cloudcreds',
    purifyFn: purifyCloudCreds,
    ctxKey: 'cloudCredss',
  },
  {
    entity: 'dockerprofiles',
    purifyFn: purifyDockerProfile,
    ctxKey: 'dockerProfiles',
  },
  {
    entity: 'users',
    purifyFn: purifyUser,
    ctxKey: 'users',
  },
  {
    entity: 'projects',
    purifyFn: purifyProject,
    ctxKey: 'projects',
  },
  {
    entity: 'scriptruntimes',
    purifyFn: purifyScriptRuntime,
    ctxKey: 'scriptRuntimes',
  },
  {
    entity: 'scripts',
    purifyFn: purifyScript,
    ctxKey: 'scripts',
  },
  {
    entity: 'applications',
    purifyFn: purifyApplication,
    ctxKey: 'applications',
  },
  {
    entity: 'datastreams',
    purifyFn: purifyDataStream,
    ctxKey: 'dataStreams',
  },
];

export function range(n: number): number[] {
  return Array(n)
    .fill(0)
    .map((v, i) => i);
}
export function randomString(len: number): string {
  const s = uuidv4();
  return s.substring(0, len);
}

export function randomAttribute(prefix: string): string {
  const suffix = randomString(10);
  const av = `${prefix}-${suffix}`;
  if (prefix === 'email') {
    return `${av}@example.com`;
  }
  return av;
}

export function randomStringArray(count: number, prefix: string): string[] {
  const p = randomAttribute(prefix);
  return range(count).map(i => `${p}-${i + 1}`);
}

export function randomCount(min, max): number {
  return min + Math.floor(Math.random() * (max - min + 1));
}

export function randomIndex(count) {
  return Math.floor(Math.random() * count);
}

export function randomIPObject() {
  const arr = Array(4)
    .fill(0)
    .map(x => randomIndex(255));
  const ipAddress = arr.join('.');
  arr[3] = 1;
  const gateway = arr.join('.');
  const subnet = '255.255.255.0';
  return {
    ipAddress,
    gateway,
    subnet,
  };
}

export function pick(choices: any[]) {
  const idx = randomIndex(choices.length);
  return choices[idx];
}

// will pick at least one
export function pickMany(choices: any[]) {
  const n = 1 + randomIndex(choices.length);
  const sources = choices.slice();
  if (n >= choices.length) {
    return sources;
  }
  const results: any[] = [];
  for (let i = 0; i < n; i++) {
    const idx = randomIndex(sources.length);
    results.push(sources.splice(idx, 1)[0]);
  }
  return results;
}

export function arrayEquals(a1: any[], a2: any[], key: string): boolean {
  if (a1.length === a2.length) {
    for (let i = 0; i < a1.length; i++) {
      if (!equal(a1[i], a2[i])) {
        console.log(`*** [${key}] not equal: a1=`, a1[i]);
        console.log(`*** [${key}] not equal: a2=`, a2[i]);
        return false;
      }
    }
    return true;
  }
  console.log(
    `*** [${key}] length not equal: ` + a1.length + ' vs ' + a2.length
  );
  return false;
}

export function verifyEntities(ctx, apiVersion: string, cats, key, purifyFn) {
  let ctxCats = ctx[key].slice(0).sort((a, b) => a.id.localeCompare(b.id));
  const pureCats = cats
    .map(c => purifyFn(c, apiVersion))
    .sort((a, b) => a.id.localeCompare(b.id));
  if (key === 'dockerProfiles') {
    // drop pwd, credentials, userName
    ctxCats = ctxCats.map(c => purifyFn(c, apiVersion));
  }
  if (key === 'cloudCredss' && ctx.name.indexOf('edge[') !== 0) {
    // mask pwd, etc
    ctxCats = ctxCats.map(c => purifyFn(c, apiVersion));
  }
  if (key === 'users') {
    // make copy to not modify original password
    ctxCats = ctxCats.map(c => {
      let { password, ...rest } = c;
      password = getSha256(password);
      // don't compare password, as password is masked in API response
      return rest;
    });
  }
  if (!arrayEquals(ctxCats, pureCats, key)) {
    console.error(`*** verify ${key} failed`);
    throw Error('failed to verify entities');
  } else {
    console.log(`>>> verified ${key} for: ${ctx.name}`);
  }
}

export function sleep(x: any, millis: number): Promise<any> {
  return new Promise(resolve => {
    setTimeout(() => resolve(x), millis);
  });
}

export function j2s(data): string {
  return JSON.stringify(data, null, 2);
}

export function wrapString(
  s: string,
  mask: string,
  start: number,
  end: number
): string {
  if (start < 0 || end < 0 || start + end === 0) {
    return s;
  }
  const n = s.length - start - end;
  const w = Array(n)
    .fill(0)
    .map(x => mask)
    .join('');
  const prefix = s.substring(0, start);
  const suffix = s.substring(s.length - end);
  return `${prefix}${w}${suffix}`;
}
