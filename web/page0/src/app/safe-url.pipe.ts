import {Pipe, PipeTransform} from '@angular/core';
import {DomSanitizer} from '@angular/platform-browser';

@Pipe({
  standalone: false,
  name: 'safeurl',
})
export class SafeUrlPipe implements PipeTransform {
  constructor(private sanitizer: DomSanitizer) {}

  transform(url: string) {
    /* eslint-disable */
    // DO NOT FORMAT THIS LINE
    return this.sanitizer.bypassSecurityTrustResourceUrl(url);
    /* eslint-enable */
  }
}
