import { Pipe, PipeTransform } from '@angular/core';
import { DecimalPipe } from '@angular/common';

@Pipe({ name: 'gb' })
export class StorageCapacityPipe implements PipeTransform {
  transform(value: number, args: string[]): any {
    if (!value) return `0 GB`;
    let unit = 'GB';
    let val = value;

    if (value > 1e15) {
      val = value / 1e15;
      unit = 'YB';
    } else if (value > 1e12) {
      val = value / 1e12;
      unit = 'ZB';
    } else if (value > 1e9) {
      val = value / 1e9;
      unit = 'EB';
    } else if (value > 1e6) {
      val = value / 1e6;
      unit = 'PB';
    } else if (value > 1e3) {
      val = value / 1e3;
      unit = 'TB';
    }
    const dp = new DecimalPipe(navigator.language);
    const v = dp.transform(val, '1.1-1');
    return `${v} ${unit}`;
  }
}
