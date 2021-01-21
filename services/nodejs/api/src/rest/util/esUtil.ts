/**
 * Convert Elastic search document to model object.
 * @param esObj Elastic search document object
 * @returns {{id: string; version: number}} model object (see model/xxx)
 */
import {
  CreateDocumentResponse,
  DeleteDocumentResponse,
  RESULT_CREATE_SUCCESS,
  RESULT_DELETE_SUCCESS,
  RESULT_NOOP,
  RESULT_UPDATE_SUCCESS,
  UpdateDocumentResponse,
} from '../model/baseModel';

export function modelFromEs(esObj) {
  return {
    id: esObj._id,
    version: esObj._version,
    ...esObj._source,
  };
}

export function modelToEs(mdlObj) {
  const { id, version, ...rest } = mdlObj;
  // must not include _id
  return rest;
}

export function isCreateSuccessful(response: CreateDocumentResponse): boolean {
  return (
    response.result === RESULT_CREATE_SUCCESS ||
    response.result === RESULT_UPDATE_SUCCESS
  );
}

export function isUpdateSuccessful(response: UpdateDocumentResponse): boolean {
  return (
    response.result === RESULT_UPDATE_SUCCESS || response.result === RESULT_NOOP
  );
}

export function isDeleteSuccessful(response: DeleteDocumentResponse): boolean {
  return response.result === RESULT_DELETE_SUCCESS;
}
