import { Component, Input, ElementRef } from '@angular/core';
import { StatusIcon } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-status-icon';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactStatusIconComponent extends PrismReactComponentBase {
  @Input() type: string;
  @Input() tooltipProps: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, StatusIcon);
  }

  protected getProps(): any {
    const { type, tooltipProps } = this;
    return {
      type,
      tooltipProps,
    };
  }
}
