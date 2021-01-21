import { BaseModel } from './baseModel';

export interface ScriptRuntime extends BaseModel {
  name: string;
  description?: string;
  language: string;
  builtin: boolean;
  dockerRepoURI?: string;
  dockerProfileID?: string;
  dockerfile?: string;
  projectId: string;
}
