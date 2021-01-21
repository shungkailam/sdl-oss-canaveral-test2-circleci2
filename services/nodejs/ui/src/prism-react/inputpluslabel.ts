import { Component, Input, ElementRef } from '@angular/core';
import { InputPlusLabel } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-input-plus-label';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactInputPlusLabelComponent extends PrismReactComponentBase {
  @Input() label: string;
  @Input() placeholder: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, InputPlusLabel);
  }

  protected getProps(): any {
    const { label, placeholder } = this;
    return {
      label,
      placeholder,
    };
  }
}
