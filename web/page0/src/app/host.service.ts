import {Injectable} from '@angular/core';
import {hostListSelector} from 'src/app/store/selectors';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {HostInstance} from 'src/app/interface/cloud-orchestrator.dto';
import {Runtime} from 'src/app/interface/runtime-interface';

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
