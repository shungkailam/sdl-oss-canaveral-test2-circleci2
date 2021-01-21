import { Component, Input, ElementRef } from '@angular/core';
import { BarChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-bar-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactBarChartComponent extends PrismReactComponentBase {
  @Input() data: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, BarChart);
  }

  protected getProps(): any {
    const { data } = this;
    return {
      data,
    };
  }
}
