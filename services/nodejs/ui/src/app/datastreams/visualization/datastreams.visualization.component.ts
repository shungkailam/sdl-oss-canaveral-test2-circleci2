import { Component, ViewChild, ElementRef, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import * as d3 from 'd3';
import { Http } from '@angular/http';
import { DataStream } from '../../model/dataStream';
import { DataSource } from '../../model/dataSource';
import {
  datastreamContainsDatasource,
  getDatasourceSensorsCount,
} from '../../utils/modelUtil';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { handleAuthError } from '../../utils/authUtil';

function link(d) {
  return (
    'M' +
    d.source.x +
    ',' +
    d.source.y +
    'C' +
    (d.source.x + d.target.x) / 2 +
    ',' +
    d.source.y +
    ' ' +
    (d.source.x + d.target.x) / 2 +
    ',' +
    d.target.y +
    ' ' +
    d.target.x +
    ',' +
    d.target.y
  );
}

@Component({
  selector: 'app-datastreams-visualization',
  templateUrl: './datastreams.visualization.component.html',
  styleUrls: ['./datastreams.visualization.component.css'],
})
export class DatastreamsVisualizationComponent implements OnInit {
  @ViewChild('chart') private chartContainer: ElementRef;
  private chart: any;

  private margin: any = { top: 0, bottom: 0, left: 0, right: 0 };
  private width: number;
  private height: number;

  datasourcesGroupBy = 'Sensor Type';
  datastreamsGroupBy = 'Location';

  badgeStyle = {
    backgroundColor: '#f6f7f8',
    color: '#9ca7b4',
    boxShadow: '0 0 0 1px #d9d9d9 inset',
    ['font-size']: '10px',
  };

  activeDataSource = null;
  activeEdgeDataStream = null;
  activeCloudDataStream = null;
  activeIndex = -1;

  constructor(private http: Http, private router: Router) {}

  datastreams = [];
  datasources = [];
  allDatasources = [];
  edgeDatastreams = [];
  cloudDatastreams = [];
  edges = [];
  edgeFilters = [];

  svg = null;
  pairs = [];

  async ngOnInit() {
    try {
      this.datastreams = await this.http
        .get('/v1/datastreams', getHttpRequestOptions())
        .toPromise()
        .then(x => x.json());
      this.datasources = await this.http
        .get('/v1/datasources', getHttpRequestOptions())
        .toPromise()
        .then(x => x.json());
      this.allDatasources = this.datasources.slice();
      this.edges = await this.http
        .get('/v1/edges', getHttpRequestOptions())
        .toPromise()
        .then(x => x.json());
      this.edgeFilters = this.edges.map(e => ({ selected: false }));
      this.edgeDatastreams = this.datastreams.filter(
        ds => ds.destination === 'Edge'
      );
      this.cloudDatastreams = this.datastreams.filter(
        ds => ds.destination === 'Cloud'
      );
      this.updateDatasourceSensorsCount();
      this.computeConnectors();
      this.createChart();
    } catch (e) {
      handleAuthError(null, e, this.router, this.http, () => this.ngOnInit());
    }
  }

  updateDatasourceSensorsCount() {
    this.datasources.forEach(ds => {
      ds.count = getDatasourceSensorsCount(ds);
      console.log(
        '>>> set dsCount of data source ' + ds.name + ' to ' + ds.count
      );
    });
  }

  createChart() {
    const element = this.chartContainer.nativeElement;
    this.width = element.offsetWidth - this.margin.left - this.margin.right;
    this.height = element.offsetHeight - this.margin.top - this.margin.bottom;
    const svg = d3
      .select(element)
      .append('svg')
      .attr('width', element.offsetWidth)
      .attr('height', element.offsetHeight);

    // chart plot area
    this.chart = svg
      .append('g')
      .attr('class', 'bars')
      .attr('transform', `translate(${this.margin.left}, ${this.margin.top})`);

    this.svg = svg;

    setTimeout(() => {
      this.renderConnectors();
    });
  }

  computeConnectors() {
    this.pairs = [];
    this.datasources.forEach((d, i) => {
      this.edgeDatastreams.forEach((ed, j) => {
        if (datastreamContainsDatasource(ed, d)) {
          this.pairs.push([`d-${d.id}`, `ds-${ed.id}`]);
        }
      });
      this.cloudDatastreams.forEach((cd, j) => {
        if (datastreamContainsDatasource(cd, d)) {
          this.pairs.push([`d-${d.id}`, `dsc-${cd.id}`]);
        }
      });
    });
    this.cloudDatastreams.forEach((cd, i) => {
      const ds = this.edgeDatastreams.find(ed => ed.id === cd.originId);
      if (ds) {
        this.pairs.push([`ds-${ds.id}`, `dsc-${cd.id}`]);
      }
    });
  }

  renderConnectors() {
    this.pairs.forEach(p => this.addPath(p));
  }

  rerenderConnectors() {
    if (this.svg) {
      const element = this.chartContainer.nativeElement;
      this.width = element.offsetWidth - this.margin.left - this.margin.right;
      this.height = element.offsetHeight - this.margin.top - this.margin.bottom;
      this.svg
        .attr('width', element.offsetWidth)
        .attr('height', element.offsetHeight);
      this.svg.selectAll('path').remove();
      this.renderConnectors();
    }
  }

  addPath(pair) {
    const e1 = document.getElementById(pair[0]);
    const e2 = document.getElementById(pair[1]);
    if (!e1 || !e2) {
      return;
    }
    const d = {
      source: {
        x: e1.offsetLeft + e1.offsetWidth,
        y: e1.offsetTop + e1.offsetHeight / 2,
      },
      target: {
        x: e2.offsetLeft,
        y: e2.offsetTop + e2.offsetHeight / 2,
      },
    };
    const highlightColor = '#22a5f7';
    let color = '#d6dbe0';
    if (this.activeDataSource) {
      if (pair[0] === `d-${this.activeDataSource.id}`) {
        color = highlightColor;
      }
    }
    if (this.activeCloudDataStream) {
      if (pair[1] === `dsc-${this.activeCloudDataStream.id}`) {
        color = highlightColor;
      }
    }
    if (this.activeEdgeDataStream) {
      if (pair[0] === `ds-${this.activeEdgeDataStream.id}`) {
        color = highlightColor;
      }
      if (pair[1] === `ds-${this.activeEdgeDataStream.id}`) {
        color = highlightColor;
      }
    }

    this.svg
      .append('path')
      .attr('stroke', color)
      .attr('fill', 'none')
      .attr('class', 'link2')
      .attr('d', function(x) {
        return link(d);
      });
  }

  onResize(event) {
    this.rerenderConnectors();
  }

  onMouseOverDataSource(event, ds, i) {
    this.activeDataSource = ds;
    this.activeIndex = i;
    this.rerenderConnectors();
  }

  onMouseOutDataSource(event, ds, i) {
    this.activeDataSource = null;
    this.activeIndex = -1;
    this.rerenderConnectors();
  }
  onMouseOverEdgeDataStream(event, ds, i) {
    this.activeEdgeDataStream = ds;
    this.activeIndex = i;
    this.rerenderConnectors();
  }

  onMouseOutEdgeDataStream(event, ds, i) {
    this.activeEdgeDataStream = null;
    this.activeIndex = -1;
    this.rerenderConnectors();
  }
  onMouseOverCloudDataStream(event, ds, i) {
    this.activeCloudDataStream = ds;
    this.activeIndex = i;
    this.rerenderConnectors();
  }

  onMouseOutCloudDataStream(event, ds, i) {
    this.activeCloudDataStream = null;
    this.activeIndex = -1;
    this.rerenderConnectors();
  }

  checkEdgeFilters(event) {
    const es = this.edges.filter((e, i) => this.edgeFilters[i].selected);
    if (es.length && es.length !== this.edges.length) {
      // filter out some datasources
      this.datasources = this.allDatasources.filter(ds =>
        es.some(e => e.id === ds.edgeId)
      );
    } else {
      // all data sources
      this.datasources = this.allDatasources.slice();
    }
    setTimeout(() => {
      this.rerenderConnectors();
    });
  }
}
