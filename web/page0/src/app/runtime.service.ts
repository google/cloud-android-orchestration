import {Injectable} from '@angular/core';
import {Observable, of} from 'rxjs';
import {mockRuntimeList} from 'src/mock/runtimes';
import {Runtime} from './runtime-interface';

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private runtimes: Observable<Runtime[]> = of(mockRuntimeList);

  getRuntimes() {
    return this.runtimes;
  }

  constructor() {}
}
