import { Component, Input, ElementRef } from '@angular/core';
import { Badge } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-badge';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactBadgeComponent extends PrismReactComponentBase {
  @Input() color: string;
  @Input() count: number;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, Badge);
  }

  protected getProps(): any {
    const { color, count } = this;
    return {
      color,
      count,
    };
  }
}
