import {Injectable} from '@angular/core';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {HostInstance} from 'src/app/interface/cloud-orchestrator.dto';
import {Runtime} from 'src/app/interface/runtime-interface';
import {of, throwError} from 'rxjs';
import {catchError, tap} from 'rxjs/operators';
import {OperationService} from './operation.service';
import {HostStatus} from './interface/host-interface';

@Injectable({
  providedIn: 'root',
})
export class HostService {
  createHost(hostInstance: HostInstance, runtime: Runtime, zone: string) {
    return this.apiService
      .createHost(runtime.url, zone, {
        host_instance: hostInstance,
      })
      .pipe(
        tap(operation => {
          const waitUrl = `${runtime.url}/v1/zones/${zone}/operations/${operation.name}`;
          this.store.dispatch({
            type: 'host-create-start',
            wait: {
              waitUrl,
              metadata: {
                type: 'host-create',
                zone,
                runtimeAlias: runtime.alias,
              },
            },
          });
          this.waitService.longPolling<HostInstance>(waitUrl).subscribe({
            next: hostInstance => {
              this.store.dispatch({
                type: 'host-create-complete',
                waitUrl,
                host: {
                  name: hostInstance.name!,
                  zone,
                  url: `${runtime.url}/v1/zones/${zone}/hosts/${hostInstance.name}`,
                  runtime: runtime.alias,
                  groups: [],
                  status: HostStatus.running,
                },
              });
            },
            error: error => {
              this.store.dispatch({
                type: 'host-create-error',
                waitUrl,
              });
              throw error;
            },
          });
        }),
        catchError(error => {
          this.store.dispatch({
            type: 'host-create-error',
          });

          return throwError(() => error);
        })
      );
  }

  deleteHost(hostUrl: string) {
    // TODO: long polling
    return this.apiService.deleteHost(hostUrl);
  }

  constructor(
    private apiService: ApiService,
    private store: Store,
    private waitService: OperationService
  ) {}
}
