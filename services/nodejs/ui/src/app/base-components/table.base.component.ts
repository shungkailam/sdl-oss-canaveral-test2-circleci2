import { OnInit, OnDestroy } from '@angular/core';
import { Router, NavigationEnd } from '@angular/router';

export class TableBaseComponent implements OnInit, OnDestroy {
  _allChecked = false;
  _indeterminate = false;
  _displayData = [];
  routerEventSubscription = null;
  routerEventUrl = null;
  _refreshDataMillis = 60000;
  _fetchTimer = null;
  _rowIndex = '';
  _dataStreamsCount = 0;
  _dataSourcesCount = 0;
  _sherlockRole = '';
  _sherlockUsername = '';
  sortName = null;
  sortValue = null;
  sortMap = {};
  popupSortMap = {};
  popupMapping = {};
  data = [];
  popupData = [];
  mapping = {};
  _totalProjects = 0;

  tableRowEditingOptions = [
    { label: 'Edit', action: () => this.onClickUpdateTableRow() },
    { label: 'View', action: () => this.onClickViewTableRow() },
    { label: 'Remove', action: () => this.onClickRemoveTableRow() },
    { label: 'Clone', action: () => this.onClickDuplicateTableRow() },
  ];
  constructor(public router: Router) {}

  _displayDataChange($event) {
    const oldData = this._displayData;
    this._displayData = $event;
    // restore previous selection
    if (oldData && oldData.length) {
      oldData.forEach(d => {
        if (d.checked) {
          const nd = this._displayData.find(x => x.id === d.id);
          if (nd) {
            nd.checked = true;
          }
        }
      });
    }
    this._refreshStatus();
  }

  _refreshStatus() {
    const allChecked = this._displayData.every(value => value.checked === true);
    const allUnChecked = this._displayData.every(value => !value.checked);
    this._allChecked = allChecked;
    this._indeterminate = !allChecked && !allUnChecked;
  }

  _checkAll(value) {
    if (value) {
      this._displayData.forEach(data => {
        data.checked = true;
      });
    } else {
      this._displayData.forEach(data => {
        data.checked = false;
      });
    }
    this._refreshStatus();
  }

  ngOnInit() {
    this.subscribeRouterEventMaybe();
    this.fetchData();
    this.checkCredentials();
    this.startFetchingData();
    this.createTableOptions();
    this.setUserName();
  }

  setUserName() {}
  checkCredentials() {
    if (localStorage['sherlock_creds']) {
      this._sherlockUsername = JSON.parse(localStorage['sherlock_creds'])[
        'username'
      ];
    }

    if (localStorage['sherlock_mynutanix_email']) {
      this._sherlockUsername = localStorage['sherlock_mynutanix_email'];
    }
    if (localStorage['sherlock_role']) {
      this._sherlockRole = localStorage['sherlock_role'];
    }
  }

  ngOnDestroy() {
    this.unsubscribeRouterEventMaybe();
    this.stopFetchingData();
  }

  subscribeRouterEventMaybe() {
    if (this.routerEventUrl) {
      this.routerEventSubscription = this.router.events.subscribe(event => {
        if (event instanceof NavigationEnd) {
          const e: NavigationEnd = event;
          if (e.url === this.routerEventUrl) {
            // refresh in case new category got created
            setTimeout(() => this.fetchData());
          }
        }
      });
    }
  }
  unsubscribeRouterEventMaybe() {
    if (this.routerEventSubscription) {
      this.routerEventSubscription.unsubscribe();
    }
  }

  startFetchingData() {
    this.stopFetchingData();
    this._fetchTimer = setInterval(
      () => this.fetchData(),
      this._refreshDataMillis
    );
  }

  stopFetchingData() {
    if (this._fetchTimer) {
      clearInterval(this._fetchTimer);
      this._fetchTimer = null;
    }
  }

  // subclass should override
  fetchData() {
    // no op
  }
  createTableOptions() {}
  onClickUpdateTableRow() {}
  onClickRemoveTableRow() {}
  onClickDuplicateTableRow() {}
  onClickViewTableRow() {}
  isShowUpdateButton() {
    if (this._displayData) {
      const checkedCount = this._displayData.reduce((acc, cur) => {
        return cur.checked ? acc + 1 : acc;
      }, 0);
      return checkedCount === 1;
    }
    return false;
  }

  sort(sortName, value) {
    this.sortName = sortName;
    this.sortValue = value;
    Object.keys(this.sortMap).forEach(key => {
      if (key !== sortName) {
        this.sortMap[key] = null;
      } else {
        this.sortMap[key] = value;
      }
    });

    this.data = this.data.slice().sort((a, b) => {
      if (a[this.mapping[this.sortName]] > b[this.mapping[this.sortName]]) {
        return this.sortValue === 'ascend' ? 1 : -1;
      } else if (
        a[this.mapping[this.sortName]] < b[this.mapping[this.sortName]]
      ) {
        return this.sortValue === 'ascend' ? -1 : 1;
      } else {
        return 0;
      }
    });
  }
  popupSort(sortName, value) {
    this.sortName = sortName;
    this.sortValue = value;
    Object.keys(this.popupSortMap).forEach(key => {
      if (key !== sortName) {
        this.popupSortMap[key] = null;
      } else {
        this.popupSortMap[key] = value;
      }
    });

    this.popupData = this.popupData.slice().sort((a, b) => {
      if (
        a[this.popupMapping[this.sortName]] >
        b[this.popupMapping[this.sortName]]
      ) {
        return this.sortValue === 'ascend' ? 1 : -1;
      } else if (
        a[this.popupMapping[this.sortName]] <
        b[this.popupMapping[this.sortName]]
      ) {
        return this.sortValue === 'ascend' ? -1 : 1;
      } else {
        return 0;
      }
    });
  }
}
