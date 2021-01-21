import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ name: 'strArray' })
export class StringArrayPipe implements PipeTransform {
  transform(value: string[], args: string[]): any {
    if (!value) {
      return '';
    }
    if (!value.length) {
      return value;
    }
    if (value.length <= 3) {
      return value.join(', ');
    }
    const prefix = value.slice(0, 3).join(', ');
    const n = value.length - 3;
    return `${prefix} and ${n} more`;
  }
}
