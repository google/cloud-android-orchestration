import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
  catchError,
  forkJoin,
  map,
  mergeScan,
  of,
  shareReplay,
  startWith,
  Subject,
  switchMap,
  tap,
} from 'rxjs';
import { HostCreateDto, HostResponseDto } from './dto';
import { Host, HostInfo } from './host-interface';
import { RuntimeService } from './runtime.service';

interface HostCreateAction {
  type: 'create';
  hostCreateDto: HostCreateDto;
  requestUrl: string;
  runtimeAlias: string;
}

interface HostDeleteAction {
  type: 'delete';
  hostUrl: string;
}

interface HostInitAction {
  type: 'init';
}

type HostAction = HostCreateAction | HostDeleteAction | HostInitAction;

@Injectable({
  providedIn: 'root',
})
export class HostService {
  private hostAction = new Subject<HostAction>();

  createHost(hostCreateDto: HostCreateDto, runtimeAlias: string) {
    return this.runtimeService.getRuntimeByAlias(runtimeAlias).pipe(
      tap((runtime) => {
        const requestUrl = `${runtime.url}`; // TODO: add zone
        console.log('requestUrl ', requestUrl);
        this.hostAction.next({
          type: 'create',
          hostCreateDto,
          requestUrl,
          runtimeAlias,
        });
      }),
      catchError((error) => {
        throw error;
      })
    );
  }

  deleteHost(hostUrl: string) {
    this.hostAction.next({ type: 'delete', hostUrl });
  }

  private hosts$ = this.hostAction.pipe(
    startWith({ type: 'init' } as HostInitAction),
    tap((action) => console.log('host: ', action)),
    mergeScan((acc, action) => {
      if (action.type === 'init') {
        return this.runtimeService.getRuntimes().pipe(
          switchMap((runtimes) => {
            return forkJoin(
              runtimes.flatMap((runtime) => {
                const hosts = runtime.hosts;
                if (!hosts) {
                  return [];
                }

                return hosts.map((hostUrl) => {
                  return this.getHostInfo(hostUrl).pipe(
                    map((info) => ({
                      name: info.name,
                      zone: info.zone,
                      runtime: runtime.alias,
                      groups: info.groups,
                      url: info.url,
                    }))
                  );
                });
              })
            );
          })
        );
      }

      if (action.type === 'create') {
        return this.postHostAPI(action.requestUrl, action.hostCreateDto).pipe(
          map((hostResponse) => {
            return [
              ...acc,
              {
                name: hostResponse.name,
                url: action.requestUrl, // TODO: construct/receive host url
                zone: hostResponse.zone,
                runtime: action.runtimeAlias,
                groups: hostResponse.groups,
              },
            ];
          }),
          catchError((error) => {
            // TODO: handle error
            console.error(error);
            return of(acc);
          })
        );
      }

      if (action.type === 'delete') {
        return this.deleteHostAPI(action.hostUrl).pipe(
          map(() => {
            return acc.filter((item) => item.url !== action.hostUrl);
          }),
          catchError((error) => {
            // TODO: handle error
            console.error(error);
            return of(acc);
          })
        );
      }

      return of(acc);
    }, [] as Host[]),
    shareReplay(1)
  );

  private getHostInfo(url: string) {
    return this.httpClient.get<HostInfo>(`${url}/info`);
  }

  private postHostAPI(requestUrl: string, host: HostCreateDto) {
    return this.httpClient.post<HostResponseDto>(`${requestUrl}`, host);
  }

  private deleteHostAPI(hostUrl: string) {
    return this.httpClient.delete<void>(hostUrl);
  }

  getHosts(runtime: string) {
    return this.hosts$.pipe(
      map((hosts) => hosts.filter((host) => host.runtime === runtime))
    );
  }

  getAllHosts() {
    return this.hosts$;
  }

  constructor(
    private httpClient: HttpClient,
    private runtimeService: RuntimeService
  ) {}
}
