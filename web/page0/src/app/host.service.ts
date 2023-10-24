import {Injectable} from '@angular/core';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {
  HostInstance,
  Operation,
} from 'src/app/interface/cloud-orchestrator.dto';
import {Runtime} from 'src/app/interface/runtime-interface';
import {Observable, throwError} from 'rxjs';
import {catchError, map, tap} from 'rxjs/operators';
import {OperationService} from './operation.service';
import {HostStatus} from './interface/host-interface';
import {Result, ResultType} from './interface/result-interface';

@Injectable({
  providedIn: 'root',
})
export class HostService {
  createHost(
    hostInstance: HostInstance,
    runtime: Runtime,
    zone: string
  ): Observable<Result<boolean>> {
    const request = this.apiService.createHost(runtime.url, zone, {
      host_instance: hostInstance,
    });

    const waitUrlSynthesizer = (op: Operation) =>
      `${runtime.url}/v1/zones/${zone}/operations/${op.name}`;

    return this.waitService
      .wait<HostInstance>(request, waitUrlSynthesizer)
      .pipe(
        map(result => {
          switch (result.type) {
            case ResultType.waitStarted:
              this.store.dispatch({
                type: 'host-create-start',
                wait: {
                  waitUrl: result.waitUrl,
                  metadata: {
                    type: 'host-create',
                    zone,
                    runtimeAlias: runtime.alias,
                  },
                },
              });
              break;
            case ResultType.done:
              const hostName = result.data.name!;
              this.store.dispatch({
                type: 'host-create-complete',
                waitUrl: result.waitUrl,
                host: {
                  name: hostName,
                  zone,
                  url: `${runtime.url}/v1/zones/${zone}/hosts/${hostName}`,
                  runtime: runtime.alias,
                  status: HostStatus.running,
                },
              });

              return {
                type: ResultType.done as ResultType.done,
                waitUrl: result.waitUrl,
                data: true,
              };
            default:
              break;
          }

          return result;
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
    this.store.dispatch({
      type: 'host-delete-start',
      wait: {
        waitUrl: hostUrl,
        metadata: {
          type: 'host-delete',
          hostUrl,
        },
      },
    });

    return this.apiService.deleteHost(hostUrl).pipe(
      tap(() => {
        this.store.dispatch({
          type: 'host-delete-complete',
          waitUrl: hostUrl,
        });
      }),
      catchError(error => {
        this.store.dispatch({
          type: 'host-delete-error',
          waitUrl: hostUrl,
        });

        return throwError(() => error);
      })
    );
  }

  constructor(
    private apiService: ApiService,
    private store: Store,
    private waitService: OperationService
  ) {}
}
