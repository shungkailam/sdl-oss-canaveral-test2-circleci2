import {
  Component,
  Input,
  Output,
  EventEmitter,
  ElementRef,
} from '@angular/core';
import { Modal } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-modal';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactModalComponent extends PrismReactComponentBase {
  @Input() title: string;
  @Input() primaryButtonLabel: string;

  @Input() visible: boolean;

  @Output() primaryButtonClick: EventEmitter<any> = new EventEmitter();

  @Output() btnCancel: EventEmitter<any> = new EventEmitter();

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, Modal);
  }

  onCancel = () => {
    this.btnCancel.emit();
  };

  onPrimaryButtonClick = () => {
    this.primaryButtonClick.emit();
  };

  protected getProps(): any {
    const {
      visible,
      onCancel,
      onPrimaryButtonClick: primaryButtonClick,
      title,
      primaryButtonLabel,
    } = this;
    return {
      visible,
      onCancel,
      primaryButtonClick,
      title,
      primaryButtonLabel,
    };
  }
}
