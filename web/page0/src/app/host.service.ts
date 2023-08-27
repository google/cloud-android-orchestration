import {Injectable} from '@angular/core';
import {of, Subject} from 'rxjs';
import {map, mergeScan, shareReplay, startWith, tap} from 'rxjs/operators';
import {hostListSelector, runtimeListSelector} from 'src/store/selectors';
import {Store} from 'src/store/store';
import {ApiService} from './api.service';
import {HostInstance} from './cloud-orchestrator.dto';
import {Host} from './host-interface';
import {Runtime} from './runtime-interface';

@Injectable({
  providedIn: 'root',
})
export class HostService {
  createHost(hostInstance: HostInstance, runtime: Runtime, zone: string) {
    // TODO: long polling
    return this.apiService.createHost(runtime.url, zone, {
      host_instance: hostInstance,
    });
  }

  deleteHost(hostUrl: string) {
    // TODO: long polling
    return this.apiService.deleteHost(hostUrl);
  }

  private hosts$ = this.store.select(hostListSelector);

  constructor(
    private apiService: ApiService,
    private store: Store
  ) {}
}
