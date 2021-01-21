import { Component, Input, ElementRef } from '@angular/core';
import { DistributionBarChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-distribution-bar-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactDistributionBarChartComponent extends PrismReactComponentBase {
  @Input() data: any;
  @Input() valueSuffix: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, DistributionBarChart);
  }

  protected getProps(): any {
    const { data, valueSuffix } = this;
    return {
      data,
      valueSuffix,
    };
  }
}
