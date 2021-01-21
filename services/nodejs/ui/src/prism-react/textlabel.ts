import { Component, Input, ElementRef } from '@angular/core';
import { TextLabel } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-text-label';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactTextLabelComponent extends PrismReactComponentBase {
  @Input() type: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, TextLabel);
  }
}
