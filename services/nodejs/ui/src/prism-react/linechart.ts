import { Component, Input, ElementRef } from '@angular/core';
import { LineChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-line-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactLineChartComponent extends PrismReactComponentBase {
  @Input() data: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, LineChart);
  }

  protected getProps(): any {
    const { data } = this;
    return {
      data,
    };
  }
}
