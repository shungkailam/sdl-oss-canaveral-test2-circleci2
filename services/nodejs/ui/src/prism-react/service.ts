import { Injectable } from '@angular/core';
import * as React from 'react';

function toDash(s) {
  return s.replace(/([A-Z])/g, function($1) {
    return '-' + $1.toLowerCase();
  });
}

export function matchAttr(name, attr): boolean {
  const a = toDash(attr);
  const b =
    name === `ng-reflect-${a}` ||
    name === `data-${a}` ||
    name === attr ||
    name === a;
  console.log('matchAttr name=' + name + ', attr=' + attr + ', return=' + b);
  return b;
}

export function divToReactElement(x) {
  const { nodeName, outerHTML } = x;
  let reactElement = null;
  if (outerHTML) {
    try {
      reactElement = React.createElement(nodeName, {
        dangerouslySetInnerHTML: {
          __html: outerHTML,
        },
      });
    } catch (e) {
      console.log('caught error:', e);
    }
  }
  return reactElement;
}

@Injectable()
export class PrismReactService {
  converters: any[] = [];
  converterKeys: string[] = [];
  converterMaps: any = {};

  constructor() {}

  registerConverter(key: string, converter: any) {
    if (!this.converterMaps[key]) {
      this.converterMaps[key] = true;
      this.converterKeys.push(key);
      this.converters.push(converter);
    }
  }

  convertNativeNodesToReactElement(nodes: any) {
    if (nodes && nodes.length) {
      return Array.prototype.filter
        .call(nodes, n => n.nodeName !== '#text' || n.textContent.trim() !== '')
        .map((x, i) => this.convertNodeToReactElement(x, i));
    }
    return null;
  }

  convertNodeToReactElement(x, key) {
    const { nodeName, outerHTML } = x;
    let reactElement = x;
    if (nodeName === '#text') {
      reactElement = x.textContent;
    } else if (outerHTML) {
      try {
        reactElement = React.createElement('div', {
          dangerouslySetInnerHTML: {
            __html: outerHTML,
          },
        });
      } catch (e) {
        console.log('caught error:', e);
      }
    }
    return reactElement;
  }
}
