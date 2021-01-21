import { Component, Input, ElementRef } from '@angular/core';
import { Table } from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-table';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactTableComponent extends PrismReactComponentBase {
  @Input() oldTable: boolean;
  @Input() dataSource: any;
  @Input() columns: any;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, Table);
  }

  protected getProps(): any {
    const { oldTable, dataSource, columns } = this;
    return {
      oldTable,
      dataSource,
      columns,
    };
  }
}
