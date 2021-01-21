import { Component, ElementRef } from '@angular/core';
import { Paragraph } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-paragraph';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactParagraphComponent extends PrismReactComponentBase {
  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, Paragraph);
  }

  protected getProps(): any {
    return {};
  }
}
