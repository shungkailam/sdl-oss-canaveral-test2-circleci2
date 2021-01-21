import { Component, OnInit, OnDestroy } from '@angular/core';
import { TableBaseComponent } from '../../../base-components/table.base.component';
import { ActivatedRoute, Router } from '@angular/router';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

@Component({
  selector: 'app-edge-edgedevices',
  templateUrl: './edge.edgedevices.component.html',
  styleUrls: ['./edge.edgedevices.component.css'],
})
export class EdgeEdgeDevicesComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  columns = ['Name', 'Performance Units', 'Storage Capacity', 'Status'];

  data = [];
  data2 = [
    {
      name: 'NTNX-IoT-Node-A',
      performanceUnits: 10,
      storageCapacity: '10.0 TB',
      status: 'Connected',
    },
    {
      name: 'NTNX-IoT-Node-B',
      performanceUnits: 10,
      storageCapacity: '10.0 TB',
      status: 'Connected',
    },
    {
      name: 'NTNX-IoT-Node-C',
      performanceUnits: 10,
      storageCapacity: '10.0 TB',
      status: 'Connected',
    },
    {
      name: 'NTNX-IoT-Node-D',
      performanceUnits: 30,
      storageCapacity: '50.0 TB',
      status: 'Connected',
    },
    {
      name: 'NTNX-IoT-Node-E',
      performanceUnits: 20,
      storageCapacity: '30.0 TB',
      status: 'Connected',
    },
  ];

  sortMap = {
    Name: null,
    'Performance Units': null,
    'Storage Capacity': null,
    Status: null,
  };

  // to resolve the naming conflict between the table title and the key from table data source
  mapping = {
    Name: 'name',
    'Performance Units': 'performanceUnits',
    'Storage Capacity': 'storageCapacity',
    status: 'status',
  };

  sub = null;
  edgeId = null;
  edge = null;

  constructor(
    router: Router,
    private http: Http,
    private route: ActivatedRoute
  ) {
    super(router);
  }

  onClickAddEdgeDevice() {
    alert('click add edge device');
  }

  onClickEntity(entity) {
    // this.router.navigate(['project', project._id]);
    alert('clicked edge device ' + entity.name);
  }

  ngOnInit() {
    this.sub = this.route.parent.params.subscribe(async params => {
      this.edgeId = params['id'];
      // fetch edge data
      this.http
        .get(`/v1/edges/${this.edgeId}`, getHttpRequestOptions())
        .toPromise()
        .then(
          x => {
            this.edge = x.json();
            this.data = this.data2.slice(0, this.edge.edgeDevices);
          },
          e =>
            handleAuthError(null, e, this.router, this.http, () =>
              this.ngOnInit()
            )
        );
    });
    super.ngOnInit();
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
  }
}
