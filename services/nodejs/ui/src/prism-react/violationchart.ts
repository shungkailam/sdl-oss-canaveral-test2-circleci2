import { Component, Input, ElementRef } from '@angular/core';
import { ViolationChart } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-violation-chart';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactViolationChartComponent extends PrismReactComponentBase {
  @Input() min: number;
  @Input() max: number;
  @Input() data: any;
  @Input() lines: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, ViolationChart);
  }

  protected getProps(): any {
    const { min, max, data, lines } = this;
    return {
      min,
      max,
      data,
      lines,
    };
  }
}
