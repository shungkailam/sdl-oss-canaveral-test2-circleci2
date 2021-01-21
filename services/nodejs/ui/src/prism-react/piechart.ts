import { Component, Input, ElementRef } from '@angular/core';
import { PieChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-pie-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactPieChartComponent extends PrismReactComponentBase {
  @Input() pies: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, PieChart);
  }

  protected getProps(): any {
    const { pies } = this;
    return {
      pies,
    };
  }
}
