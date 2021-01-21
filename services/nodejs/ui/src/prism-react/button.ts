import { Component, Output, EventEmitter, ElementRef } from '@angular/core';
import { Button } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-button';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactButtonComponent extends PrismReactComponentBase {
  @Output() btnClick: EventEmitter<any> = new EventEmitter();

  onClick = () => {
    this.btnClick.emit();
  };

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, Button);
  }

  protected getProps(): any {
    const { onClick } = this;
    return {
      onClick,
    };
  }
}
