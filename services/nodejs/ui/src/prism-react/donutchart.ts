import { Component, Input, ElementRef } from '@angular/core';
import { DonutChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-donut-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactDonutChartComponent extends PrismReactComponentBase {
  @Input() data: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, DonutChart);
  }

  protected getProps(): any {
    const { data } = this;
    return {
      data,
    };
  }
}
