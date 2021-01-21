import { TableBaseComponent } from './table.base.component';
import { Router } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { handleAuthError } from '../utils/authUtil';

function catsContains(cats, subCats) {
  if (!cats || !subCats) {
    return false;
  }
  return subCats.every(sc =>
    cats.some(c => c.id === sc.id && c.value === sc.value)
  );
}

function dataStreamContainsDataSource(dataStream, dataSource) {
  if (dataStream.origin === 'Data Source') {
    return catsContains(dataStream.originSelectors, dataSource.selectors);
  }
  return false;
}

export class DataSourceBaseComponent extends TableBaseComponent {
  data = [];
  dataStreams = [];
  isLoading = false;
  edges = [];

  constructor(router: Router, public http: Http) {
    super(router);
  }

  async fetchData() {
    this.isLoading = true;
    try {
      const data = await this.fetchDataSources();
      this.dataStreams = await this.fetchDataStreams();
      this.edges = await this.fetchEdges();
      data.sort((a, b) => a.name.localeCompare(b.name));
      this.data = data;
      this.onFetchData();
    } catch (e) {
      handleAuthError(null, e, this.router, this.http, () => this.fetchData());
    }
    this.isLoading = false;
  }

  onFetchData() {}

  // subclass to override
  async fetchDataSources() {
    return [];
  }
  fetchEdges() {
    return this.http
      .get('/v1/edges', getHttpRequestOptions())
      .toPromise()
      .then(
        x => x.json(),
        e =>
          handleAuthError(null, e, this.router, this.http, () =>
            this.fetchData()
          )
      );
  }

  fetchDataStreams() {
    return this.http
      .get('/v1/datastreams', getHttpRequestOptions())
      .toPromise()
      .then(
        x => x.json(),
        e =>
          handleAuthError(null, e, this.router, this.http, () =>
            this.fetchData()
          )
      );
  }

  isShowingDeleteButton() {
    if (
      (this._indeterminate || this._allChecked) &&
      this._displayData.length !== 0
    ) {
      const dataStreams = this.dataStreams;
      // don't allow delete if a selected data source is the origin of some stream
      // return !this._displayData.some(
      //   dsrc =>
      //     dsrc.checked &&
      //     dataStreams.some(stream => dataStreamContainsDataSource(stream, dsrc))
      // );
      return true;
    }
    return false;
  }
}
