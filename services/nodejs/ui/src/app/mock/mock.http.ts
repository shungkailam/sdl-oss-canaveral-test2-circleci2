import { Injectable } from '@angular/core';
import {
  XHRBackend,
  ConnectionBackend,
  RequestOptions,
  Request,
  RequestOptionsArgs,
  Response,
  ResponseOptions,
  Http,
} from '@angular/http';
import { Observable } from 'rxjs/Rx';

import {
  MOCK_EDGES,
  MOCK_SCRIPTS,
  MOCK_CATEGORITES,
  MOCK_DATASOURCES,
  MOCK_DATASTREAMS,
  MOCK_EDGE_DATASOURCES,
} from './mock.data';

@Injectable()
export class MockHttp extends Http {
  constructor(backend: ConnectionBackend, defaultOptions: RequestOptions) {
    super(backend, defaultOptions);
  }

  request(
    url: string | Request,
    options?: RequestOptionsArgs
  ): Observable<Response> {
    return super.request(url, options);
  }

  get(url: string, options?: RequestOptionsArgs): Observable<Response> {
    let body = null;
    switch (url) {
      case '/v1/edges':
        body = MOCK_EDGES;
        break;
      case '/v1/categories':
        body = MOCK_CATEGORITES;
        break;
      case '/v1/scripts':
        body = MOCK_SCRIPTS;
        break;
      case '/v1/datasources':
        body = MOCK_DATASOURCES;
        break;
      case '/v1/datastreams':
        body = MOCK_DATASTREAMS;
        break;
      default:
        break;
    }
    if (url.match(/\/v1\/edges\/[A-Z]+\/datasources/)) {
      body = MOCK_EDGE_DATASOURCES;
    }

    if (body) {
      return new Observable<Response>(subscribe => {
        subscribe.next(
          new Response(
            new ResponseOptions({
              body,
            })
          )
        );
        subscribe.complete();
      });
    }
    return super.get(url, options);
  }

  post(
    url: string,
    body: any,
    options?: RequestOptionsArgs
  ): Observable<Response> {
    let resp: any = null;
    if (url === '/v1/common/aggregates') {
      if (
        body.field === 'transformationArgsList' &&
        body.type === 'datastream'
      ) {
        resp = [];
      }
      if (body.field === 'edgeId' && body.type === 'datasource') {
        resp = [];
      }
    } else if (url === '/v1/common/nestedAggregates') {
      if (
        body.type === 'datastream' &&
        body.field === 'originSelectors' &&
        body.nestedField === 'id'
      ) {
        resp = [{ key: 'AV9FRUpqDKNJ0446Npi6', doc_count: 5 }];
      }
      if (
        body.type === 'datasource' &&
        body.field === 'selectors' &&
        body.nestedField === 'id'
      ) {
        resp = [
          { key: 'AV9FRUp1DKNJ0446Npi7', doc_count: 132 },
          { key: 'AV9FRUpqDKNJ0446Npi6', doc_count: 132 },
        ];
      }
    }
    if (resp) {
      return new Observable<Response>(subscribe => {
        subscribe.next(
          new Response(
            new ResponseOptions({
              body: resp,
            })
          )
        );
        subscribe.complete();
      });
    }
    return super.post(url, body, options);
  }

  put(
    url: string,
    body: string,
    options?: RequestOptionsArgs
  ): Observable<Response> {
    return super.put(url, body, options);
  }

  delete(url: string, options?: RequestOptionsArgs): Observable<Response> {
    return super.delete(url, options);
  }
}

export function httpFactory(
  xhrBackend: XHRBackend,
  requestOptions: RequestOptions
): Http {
  return new MockHttp(xhrBackend, requestOptions);
}
