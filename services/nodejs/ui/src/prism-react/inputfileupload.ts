import { Component, Input, ElementRef } from '@angular/core';
import { InputFileUpload } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-input-file-upload';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactInputFileUploadComponent extends PrismReactComponentBase {
  @Input() type: string;
  @Input() message: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, InputFileUpload);
  }

  protected getProps(): any {
    const { type, message } = this;
    return {
      type,
      message,
    };
  }
}
