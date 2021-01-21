import { Component, Input, ElementRef } from '@angular/core';
import { Alert } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-alert';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactAlertComponent extends PrismReactComponentBase {
  @Input() type: string;
  @Input() message: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, Alert);
  }

  protected getProps(): any {
    const { type, message } = this;
    return {
      type,
      message,
    };
  }
}
