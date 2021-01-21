import { Component, Input, ElementRef } from '@angular/core';
import { SparkLine } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-spark-line';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactSparkLineComponent extends PrismReactComponentBase {
  @Input() data: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, SparkLine);
  }

  protected getProps(): any {
    const { data } = this;
    return {
      data,
    };
  }
}
