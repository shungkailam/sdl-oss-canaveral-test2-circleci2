// script to help create tenant mock data

import {
  Tenant,
  User,
  Edge,
  Category,
  Script,
  DataSource,
  DataStream,
  CloudCreds,
  BaseModel,
  Application,
  ApplicationStatus,
  DockerProfile,
  ScriptRuntime,
  Project,
  EdgeInfo,
} from '../model/index';
import { DocType } from '../model/baseModel';

import { getDBService } from '../db-configurator/dbConfigurator';
import { Sequelize } from 'sequelize';
import { QueryTypes } from 'sequelize';
import { staggerPromises } from './common';

const dbService = getDBService();
const { createDocument, createAllTables } = dbService;
import { CreateDocumentResponse } from '../model/baseModel';
import { createBuiltinScriptRuntimes } from './scriptRuntimeHelper';
import {
  createDataTypeCategory,
  addDataTypeCategorySelectors,
} from './categoryHelper';

import { createDefaultProject } from '../../scripts/common';
import { createTenantRootCA } from '../../getCerts/getCerts';

let gSql: Sequelize = null;

function range(n) {
  return Array(n)
    .fill(0)
    .map((x, i) => i);
}

// NOTE: this function assumes gSql is set,
// which is done in TenantBuilder constructor.
export async function createDocNew(
  tenantId: string,
  docType: DocType,
  doc
): Promise<CreateDocumentResponse> {
  console.log(`>>> createDocNew: docType=${docType}, doc=`, doc);
  if (docType === DocType.Category) {
    // special handling for Category.values
    const { values, ...rest } = <any>doc;
    const resp = await createDocument(doc.id, docType, rest);
    const categoryId = resp._id;
    const allPS = await Promise.all(
      values.map(value =>
        createDocument(doc.id, DocType.CategoryValue, {
          categoryId,
          value,
        })
      )
    );
    return resp;
  } else if (docType === DocType.DataStream) {
    // Special handling for DataStream.originSelectors
    const { originSelectors, ...rest } = <any>doc;
    const resp = await createDocument(doc.id, docType, rest);
    const dataStreamId = resp._id;
    const originSelectorCategoryValueIds = (await Promise.all(
      originSelectors.map(os =>
        gSql.query(
          `SELECT id from category_value_model WHERE category_id ='${
            os.id
          }' AND value='${os.value}'`
        )
      )
    )).map(x => parseInt(x[1].rows[0].id, 10));
    await Promise.all(
      originSelectors.map((os, i) => {
        const categoryValueId = originSelectorCategoryValueIds[i];
        console.log(
          'creating DataStreamOrigin document, dataStreamId=' +
            dataStreamId +
            ', categoryValueId=' +
            categoryValueId
        );
        return createDocument(doc.id, DocType.DataStreamOrigin, {
          dataStreamId,
          categoryValueId,
        });
      })
    );
    return resp;
  } else if (docType === DocType.DataSource) {
    // special handling for DataSource. fields and selectors
    const { fields, selectors, ...rest } = <any>doc;
    const resp = await createDocument(doc.id, docType, rest);
    const dataSourceId = resp._id;
    // fields
    const fieldGenP = field => {
      const { name, mqttTopic, fieldType } = field;
      return createDocument(doc.id, DocType.DataSourceField, {
        dataSourceId,
        name,
        mqttTopic,
        fieldType,
      });
    };
    await staggerPromises(fields, fieldGenP, 4, null);
    // selectors
    addDataTypeCategorySelectors(doc.tenantId, fields, selectors);
    const fieldSelectorCategoryValueIds = (await Promise.all(
      selectors.map(fs =>
        gSql.query(
          `SELECT id from category_value_model WHERE category_id ='${
            fs.id
          }' AND value='${fs.value}'`
        )
      )
    )).map(x => parseInt(x[1].rows[0].id, 10));
    // scope: __ALL__ or fieldName
    // fs -> fs.scopes -> [id for name]
    const selectorFieldIdsGenP = async fs => {
      const scope = fs.scope;
      if (scope.length === 1 && scope[0] === '__ALL__') {
        return (await (<any>(
          gSql.query(
            `SELECT id from data_source_field_model WHERE data_source_id ='${dataSourceId}'`
          )
        )))[1].rows.map(r => parseInt(r.id, 10));
      } else {
        return (await Promise.all(
          scope.map(fn =>
            gSql.query(
              `SELECT id from data_source_field_model WHERE data_source_id ='${dataSourceId}' AND name='${fn}'`
            )
          )
        )).map(x => parseInt(x[1].rows[0].id, 10));
      }
    };
    const fieldSelectorFieldIds: any[] = await staggerPromises(
      selectors,
      selectorFieldIdsGenP,
      4,
      null
    );
    const selectorGenP = i =>
      Promise.all(
        fieldSelectorFieldIds[i].map(fieldId => {
          const categoryValueId = fieldSelectorCategoryValueIds[i];
          return createDocument(doc.id, DocType.DataSourceFieldSelector, {
            dataSourceId,
            categoryValueId,
            fieldId,
          });
        })
      );
    await staggerPromises(
      range(fieldSelectorFieldIds.length),
      selectorGenP,
      4,
      null
    );
    return resp;
  } else if (docType === DocType.Project) {
    // special handling for cloudCredentialIds, dockerProfileIds, users

    const { cloudCredentialIds, dockerProfileIds, users, ...rest } = <any>doc;
    const resp = await createDocument(doc.id, docType, rest);
    const projectId = resp._id;
    const allProjectDockerProfiles = await Promise.all(
      dockerProfileIds.map(dockerProfileId =>
        createDocument(projectId, DocType.ProjectDockerProfile, {
          projectId,
          dockerProfileId,
        })
      )
    );
    console.log(
      'createDocNew: got allProjectDockerProfiles:',
      allProjectDockerProfiles
    );
    const allProjectCloudCreds = await Promise.all(
      cloudCredentialIds.map(cloudCredsId =>
        createDocument(projectId, DocType.ProjectCloudCreds, {
          projectId,
          cloudCredsId,
        })
      )
    );
    console.log(
      'createDocNew: got allProjectCloudCreds:',
      allProjectCloudCreds
    );
    const allProjectUsers = await Promise.all(
      users.map(({ userId, role }) =>
        createDocument(projectId, DocType.ProjectUser, {
          projectId,
          userId,
          role,
        })
      )
    );
    console.log('createDocNew: got allProjectUsers:', allProjectUsers);
    return resp;
  } else if (docType === DocType.Application) {
    const resp = await createDocument(doc.id, docType, doc);
    // add all edges to application
    if (doc.edgeIds && doc.edgeIds.length) {
      await Promise.all(
        doc.edgeIds.map(id =>
          gSql.query(
            `INSERT INTO application_edge_model (application_id, edge_id) VALUES ('${
              doc.id
            }', '${id}')`,
            { type: QueryTypes.INSERT }
          )
        )
      );
    }

    return resp;
  } else {
    return createDocument(doc.id, docType, doc);
  }
}

function createDocs<T extends BaseModel>(
  docs: T[],
  docType: DocType
): Promise<boolean> {
  return new Promise<boolean>((resolve, reject) => {
    const n = docs.length;
    let rc = 0;
    let failCount = 0;

    const doneCheck = () => {
      if (rc === n) {
        resolve(failCount === 0);
      }
    };

    if (n) {
      docs.forEach(async doc => {
        try {
          await createDocNew(doc.id, docType, doc);
          rc += 1;
          doneCheck();
        } catch (e) {
          // ignore
          console.log(`createDocs - failed to create one doc ${docType}`, doc);
          console.log('exception: ', e);
          rc += 1;
          failCount += 1;
          doneCheck();
        }
      });
    } else {
      resolve(true);
    }
  });
}
export class TenantBuilder {
  categories: Category[] = [];
  edges: Edge[] = [];
  users: User[] = [];
  scripts: Script[] = [];
  dataSources: DataSource[] = [];
  dataStreams: DataStream[] = [];
  cloudProfiles: CloudCreds[] = [];
  applications: Application[] = [];
  applicationsStatus: ApplicationStatus[] = [];
  dockerProfiles: DockerProfile[] = [];
  scriptRuntimes: ScriptRuntime[] = [];
  projects: Project[] = [];
  edgeInfos: EdgeInfo[] = [];

  created = false;

  constructor(private tenant: Tenant, private sql: Sequelize) {
    gSql = sql;
  }
  addCategory(category: Category) {
    if (this.created) {
      throw Error('addCategory: called after create');
    }
    this.categories.push(category);
    return this;
  }
  addEdge(edge: Edge) {
    if (this.created) {
      throw Error('addEdge: called after create');
    }
    this.edges.push(edge);
    return this;
  }
  addUser(user: User) {
    if (this.created) {
      throw Error('addUser: called after create');
    }
    this.users.push(user);
    return this;
  }
  addScript(script: Script) {
    if (this.created) {
      throw Error('addScript: called after create');
    }
    this.scripts.push(script);
    return this;
  }
  addDataSource(dataSource: DataSource) {
    if (this.created) {
      throw Error('addDataSource: called after create');
    }
    this.dataSources.push(dataSource);
    return this;
  }
  addDataStream(dataStream: DataStream) {
    if (this.created) {
      throw Error('addDataStream: called after create');
    }
    this.dataStreams.push(dataStream);
    return this;
  }
  addCloudProfile(cloudCreds: CloudCreds) {
    if (this.created) {
      throw Error('addCloudProfile: called after create');
    }
    this.cloudProfiles.push(cloudCreds);
    return this;
  }

  addApplication(application: Application) {
    if (this.created) {
      throw Error('addApplication: called after create');
    }
    this.applications.push(application);
    return this;
  }
  addApplicationStatus(applicationStatus: ApplicationStatus) {
    if (this.created) {
      throw Error('addApplicationStatus: called after create');
    }
    this.applicationsStatus.push(applicationStatus);
    return this;
  }
  addDockerProfile(dockerProfile: DockerProfile) {
    if (this.created) {
      throw Error('addDockerProfile: called after create');
    }
    this.dockerProfiles.push(dockerProfile);
    return this;
  }
  addScriptRuntime(scriptRuntime: ScriptRuntime) {
    if (this.created) {
      throw Error('addScriptRuntime: called after create');
    }
    this.scriptRuntimes.push(scriptRuntime);
    return this;
  }
  addProject(project: Project) {
    if (this.created) {
      throw Error('addProject: called after create');
    }
    this.projects.push(project);
    return this;
  }
  addEdgeInfo(edgeInfo: EdgeInfo) {
    if (this.created) {
      throw Error('addEdgeInfo: called after create');
    }
    this.edgeInfos.push(edgeInfo);
    return this;
  }

  // pre-condition: SQL DB must have been initialized
  create(): Promise<boolean> {
    this.created = true;
    return new Promise<boolean>(async (resolve, reject) => {
      try {
        let success = true;
        // create DB tables
        await createAllTables();
        // create tenant
        await createDocument(this.tenant.id, DocType.Tenant, this.tenant);

        // Create a root CA for this tenant
        // example ROOT_CA_URL: https://cfssl-test.ntnxsherlock.com/rootca
        // This env var is optional and not needed when running in cloudmgmt pod.
        // Note: ROOT_CA_URL requires cfssl server endpoint to be exposed via Route53,
        // as such it currently only works for test namespace.
        try {
          await createTenantRootCA(this.tenant.id, process.env.ROOT_CA_URL);
        } catch (ex) {
          console.warn('Failed to create root CA, ignored', ex);
        }

        // add (edges, users, cloud profiles, docker profiles) first so default project will pick them up
        success = success && (await createDocs(this.edges, DocType.Edge));
        success = success && (await createDocs(this.users, DocType.User));

        success =
          success && (await createDocs(this.cloudProfiles, DocType.CloudCreds));

        success =
          success &&
          (await createDocs(this.dockerProfiles, DocType.DockerProfile));

        // create default project
        console.log('creating default project for ' + this.tenant.id);
        await createDefaultProject(gSql, this.tenant.id, false);

        // now create builtin script runtimes
        success =
          success &&
          !!(await createBuiltinScriptRuntimes(this.sql, this.tenant.id));

        // create data type category
        success =
          success && !!(await createDataTypeCategory(this.sql, this.tenant.id));

        success =
          success && (await createDocs(this.categories, DocType.Category));

        success =
          success && (await createDocs(this.edgeInfos, DocType.EdgeInfo));

        success = success && (await createDocs(this.projects, DocType.Project));

        success =
          success && (await createDocs(this.dataSources, DocType.DataSource));

        success =
          success &&
          (await createDocs(this.scriptRuntimes, DocType.ScriptRuntime));

        success = success && (await createDocs(this.scripts, DocType.Script));

        success =
          success && (await createDocs(this.dataStreams, DocType.DataStream));

        success =
          success && (await createDocs(this.applications, DocType.Application));

        success =
          success &&
          (await createDocs(
            this.applicationsStatus,
            DocType.ApplicationStatus
          ));

        resolve(success);
      } catch (e) {
        reject(e);
      }
    });
  }
}
