import { BaseModel, BaseModelKeys, ScriptParam } from './baseModel';

/**
 * Script represent lambdas:
 * functions or transformations that can be applied
 * to DataStreams.
 * Scripts are tenant-wide objects and the same script
 * may be run within an edge, across all edges of a tenant
 * or on tenant data in the cloud.
 */
export interface Script extends BaseModel {
  /**
   * name of the script
   */
  name: string;
  /**
   * name for the script
   */
  description?: string;
  /**
   * type of the script.
   * A Transformation takes a DataStream (Kafka topic) as input and produces
   * another DataStream (Kafka topic) as output.
   * A Function takes a DataStream as input,
   * but has no constraint on output.
   */
  type: 'Transformation' | 'Function';
  /**
   * Programming langulage the code is written in.
   * Supported languages are (see fission.io):
   * go
   * javascript
   * perl
   * python 3
   * php 7
   * ruby
   */
  language: string;
  /**
   * Environment for the script to run in.
   * In addition to default environment provided by fission,
   * we will also create some custom environments. E.g., we will
   * have our custom python env with tensorflow libs, etc.
   */
  environment: string;
  /**
   * The source code for the script.
   * Post .NEXT Nice we will extend this to support containers as well.
   */
  code: string;

  /**
   * Array of script parameters.
   */
  params: ScriptParam[];

  runtimeId: string;

  runtimeTag?: string;

  projectId: string;

  builtin: boolean;
  // note: we don't keep references of DataStreams that use this script here
  // instead, the references are stored in DataStreams.
  // In UI, we show which DataStreams are using this script -
  // to answer that requires an aggregate search query
}
export const ScriptKeys = [
  'name',
  'type',
  'language',
  'environment',
  'code',
  'params',
].concat(BaseModelKeys);
