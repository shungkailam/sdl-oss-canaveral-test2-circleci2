import { Component, OnDestroy, OnInit } from '@angular/core';

@Component({
  selector: 'app-prism-react-test',
  templateUrl: './prism-react-test.component.html',
  styleUrls: ['./prism-react-test.component.css'],
})
export class PrismReactTestComponent implements OnInit, OnDestroy {
  modalVisible = false;

  columns = [
    {
      title: 'Name',
      key: 'name',
    },
    {
      title:
        'Age (This is just testing to see how it would look with a long title)',
      key: 'age',
    },
    {
      title: 'Address',
      key: 'address',
    },
  ];

  data = [
    {
      key: '1',
      name: 'John Brown',
      age: 32,
      address: 'New York No. 1 Lake Park',
    },
    {
      key: '2',
      name: 'Jim Green',
      age: 42,
      address: 'London No. 1 Lake Park',
    },
    {
      key: '3',
      name: 'Joe Black',
      age: 32,
      address: 'Sidney No. 1 Lake Park',
    },
  ];

  oldTable = false;

  chartData = [
    { name: 'Page A', value: 4000, pv: 2400, amt: 2400 },
    { name: 'Page B', value: 3000, pv: 1398, amt: 2210 },
    { name: 'Page C', value: 2000, pv: 9800, amt: 2290 },
    { name: 'Page D', value: 2780, pv: 3908, amt: 2000 },
    { name: 'Page E', value: 1890, pv: 4800, amt: 2181 },
    { name: 'Page F', value: 2390, pv: 3800, amt: 2500 },
    { name: 'Page G', value: 3490, pv: 4300, amt: 2100 },
  ];

  barchartData = [
    { name: 'Page A', value: 40, label: '4000 $ Label' },
    { name: 'Page B', value: 30, label: '3000 $ Label' },
    { name: 'Page C', value: 20, label: '2000 $ Label' },
    { name: 'Page D', value: 27 },
    { name: 'Page E', value: 18 },
    { name: 'Page F', value: 23 },
    { name: 'Page G', value: 34 },
  ];
  barchartBillingData = [
    {
      day: 'Mon',
      date: '17',
      chargeAmount: 200,
      displayAmount: '$200',
    },
    {
      day: 'Tues',
      date: '18',
      chargeAmount: 150,
      displayAmount: '$150',
    },
    {
      day: 'Wed',
      date: '19',
      chargeAmount: 25,
      displayAmount: '$25',
    },
    {
      day: 'Thurs',
      date: '20',
      chargeAmount: 50,
      displayAmount: '$50',
    },
    {
      day: 'Fri',
      date: '21',
      chargeAmount: 80,
      displayAmount: '$80',
    },
  ];

  distributionBarchartData = [
    {
      name: 'Ext 4',
      value: 14.82,
    },
    {
      name: 'Nutanix Home',
      value: 1.88,
    },
    {
      name: 'Ext 4',
      value: 1.79,
    },
    {
      name: 'Content Cache',
      value: 1.21,
    },
    {
      name: 'Genesis',
      value: 0.82,
    },
    {
      name: 'Cassandra',
      value: 0.82,
    },
    {
      name: 'Curator',
      value: 0.21,
    },
    {
      name: 'Opologo',
      value: 0.14,
    },
  ];

  donutChartData = [
    { name: 'DR Untestsed', value: 100 },
    { name: 'DR Tested but Unconfigured', value: 250 },
    { name: 'DR Configured', value: 200 },
  ];

  lineChartData = [
    {
      name: 'A',
      value: 200,
    },
    {
      name: 'B',
      value: 500,
    },
    {
      name: 'C',
      value: 1000,
    },
  ];

  pieChartData = [
    {
      data: [
        { name: 'A1', b: 'a1', value: 100 },
        { name: 'A2', b: 'a2', value: 300 },
        { name: 'B1', b: 'a3', value: 100 },
        { name: 'B2', b: 'a4', value: 80 },
        { name: 'B3', b: 'a5', value: 40 },
      ],
      dataKey: 'value',
      nameKey: 'b',
    },
  ];

  sparklineData = Array(20)
    .fill(0)
    .map(x => ({ name: x, value: Math.floor(Math.random() * 50) }));

  vcmin = 400;
  vcmax = 1000;
  vclines = [
    {
      dataKey: 'Queued IOPS',
      stroke: '#DF4A53',
    },
    {
      dataKey: 'Delivered IOPS',
      stroke: '#2282E3',
    },
  ];

  vcdata = [
    { name: '01:00', value: 700 },
    { name: '02:00', value: 900 },
    { name: '03:00', value: 1100 },
    { name: '04:00', value: 1300 },
    { name: '05:00', value: 900 },
    { name: '06:00', value: 800 },
    { name: '07:00', value: 700 },
    { name: '08:00', value: 700 },
    { name: '09:00', value: 900 },
    { name: '10:00', value: 1000 },
    { name: '11:00', value: 1200 },
    { name: '12:00', value: 1500 },
    { name: '13:00', value: 1550 },
    { name: '14:00', value: 700 },
    { name: '15:00', value: 700 },
    { name: '16:00', value: 300 },
    { name: '17:00', value: 1300 },
    { name: '18:00', value: 1200 },
  ];

  ngOnInit() {
    //
  }

  ngOnDestroy() {
    //
  }

  onClickReactButton() {
    this.modalVisible = true;
  }
  onClickReactButton2() {
    alert('click react button 2 in edges');
  }
  onModalDone() {
    this.modalVisible = false;
  }
  onModalCancel() {
    this.modalVisible = false;
  }
}
