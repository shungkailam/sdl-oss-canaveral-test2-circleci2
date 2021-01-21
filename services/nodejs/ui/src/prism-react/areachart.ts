import { Component, Input, ElementRef } from '@angular/core';
import { AreaChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-area-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactAreaChartComponent extends PrismReactComponentBase {
  @Input() data: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, AreaChart);
  }

  protected getProps(): any {
    const { data } = this;
    return {
      data,
    };
  }
}
