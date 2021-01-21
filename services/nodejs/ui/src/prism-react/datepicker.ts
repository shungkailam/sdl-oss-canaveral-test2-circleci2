import { Component, Input, ElementRef } from '@angular/core';
import { DatePicker } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-date-picker';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactDatePickerComponent extends PrismReactComponentBase {
  @Input() placeholder: string;
  @Input() renderPosition: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, DatePicker);
  }

  protected getProps(): any {
    const { placeholder, renderPosition } = this;
    return {
      placeholder,
      renderPosition,
    };
  }
}
