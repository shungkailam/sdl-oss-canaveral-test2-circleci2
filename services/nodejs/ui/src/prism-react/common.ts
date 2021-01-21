import {
  OnInit,
  OnDestroy,
  OnChanges,
  AfterViewInit,
  ContentChild,
  ElementRef,
} from '@angular/core';

import * as React from 'react';
import * as ReactDOM from 'react-dom';

import { PrismReactService } from './service';

export const PrismReactComponentTemplate = `<ng-content></ng-content>`;

export class PrismReactComponentBase
  implements OnInit, OnDestroy, OnChanges, AfterViewInit {
  @ContentChild('content') child;

  mounted = false;

  constructor(
    public prismReactService: PrismReactService,
    public elRef: ElementRef,
    public reactComponent: any // React.Component
  ) {}

  protected getRootDomNode() {
    return this.elRef.nativeElement;
  }

  log(prefix: string) {
    const name = this.getName();
    console.log(`${name}: ${prefix}: child:`, this.child);
  }

  isMounted(): boolean {
    return this.mounted;
  }

  ngOnInit() {
    this.mounted = true;
  }

  ngOnChanges() {
    this.render();
  }

  ngAfterViewInit() {
    this.render();
  }

  ngOnDestroy() {
    ReactDOM.unmountComponentAtNode(this.getRootDomNode());
  }

  protected render() {
    if (this.isMounted()) {
      this.log(`render {`);
      ReactDOM.render(
        React.createElement(
          this.getReactComponent(),
          this.getProps(),
          this.getChildren()
        ),
        this.getRootDomNode()
      );
      this.log(`render }`);
    }
  }

  protected getName() {
    return this.reactComponent ? this.reactComponent.name : '<base>';
  }

  protected getReactComponent() {
    return this.reactComponent;
  }

  protected getChildren() {
    if (this.child && this.child.nativeElement) {
      return this.prismReactService.convertNativeNodesToReactElement(
        this.child.nativeElement.childNodes
      );
    }
    return null;
  }

  // subclass to override
  protected getProps(): any {
    return {};
  }
}
