import { Component, Input, ElementRef } from '@angular/core';
import { BarChartBilling } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-bar-chart-billing';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactBarChartBillingComponent extends PrismReactComponentBase {
  @Input() data: any;
  @Input() width: number;
  @Input() height: number;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, BarChartBilling);
  }

  protected getProps(): any {
    const { data, width, height } = this;
    return {
      data,
      width,
      height,
    };
  }
}
