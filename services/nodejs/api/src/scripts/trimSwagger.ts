import * as fs from 'fs';

function trimSwaggerFile(filePath) {
  return new Promise((resolve, reject) => {
    fs.readFile(filePath, 'utf8', function(err, contents) {
      // console.log(contents);
      if (err) {
        reject(err);
      } else {
        try {
          const newContent = trimSwagger(contents);
          fs.writeFile(filePath, newContent, function(err2) {
            if (err2) {
              reject(err2);
            } else {
              resolve();
            }
          });
        } catch (e) {
          reject();
        }
      }
    });
  });
}

function trimSwagger(content: string) {
  const cj = JSON.parse(content);
  cj.definitions = trimDefinitions(cj.definitions);
  cj.paths = trimPaths(cj.paths);
  return JSON.stringify(cj, null, 2);
}
// remove all ntnx:ignore properties from all definition objects
function trimDefinitions(dfns: any) {
  const nd: any = {};
  Object.keys(dfns).forEach(k => {
    const v = dfns[k];
    if (v.properties) {
      v.properties = trimProperties(v.properties, v);
    }
    nd[k] = v;
  });
  return nd;
}
function trimProperties(props: any, ctx: any) {
  const np: any = {};
  Object.keys(props).forEach(p => {
    const v = props[p];
    if (!v.description || v.description.indexOf('ntnx:ignore') === -1) {
      np[p] = v;
    } else {
      if (ctx.required) {
        const i = ctx.required.indexOf(p);
        if (i !== -1) {
          ctx.required.splice(i, 1);
        }
      }
    }
  });
  return np;
}

function trimPaths(paths: any) {
  const nps: any = {};
  Object.keys(paths).forEach(p => {
    // remove all paths starting with /wsdocs/
    if (p.indexOf('/wsdocs/') === -1) {
      const path = paths[p];
      const pops: any = {};
      let empty = true;
      // filter out ops whose summary contains ntnx:ignore
      Object.keys(path).forEach(opKey => {
        const op = path[opKey];
        if (!op.summary || op.summary.indexOf('ntnx:ignore') === -1) {
          pops[opKey] = op;
          empty = false;
        }
      });
      if (!empty) {
        nps[p] = pops;
      }
    }
  });
  return nps;
}

const USAGE = `\nUsage: node trimSwagger.js <path to swagger json file>\n`;
async function main() {
  if (process.argv.length < 3) {
    console.log(USAGE);
    process.exit(1);
  }
  const jsonPath = process.argv[2];
  try {
    await trimSwaggerFile(jsonPath);
  } catch (e) {
    console.error('Failed to trim swagger file:', e);
    process.exit(1);
  }
}

main();
