import {Injectable} from '@angular/core';
import {Observable, of, forkJoin, merge} from 'rxjs';
import {
  map,
  shareReplay,
  tap,
  withLatestFrom,
  switchMap,
  catchError,
  mergeMap,
  defaultIfEmpty,
  mergeAll,
} from 'rxjs/operators';
import {Store} from 'src/store/store';
import {ApiService} from './api.service';
import {Host} from './host-interface';
import {Runtime, RuntimeViewStatus, RuntimeStatus} from './runtime-interface';
import {defaultRuntimeSettings} from './settings';

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private getStoredRuntimes(): Runtime[] {
    const runtimes = window.localStorage.getItem('runtimes');
    // TODO: handle type error
    if (runtimes) {
      return JSON.parse(runtimes) as Runtime[];
    }

    return [];
  }

  private getInitRuntimeSettings() {
    const storedRuntimes = this.getStoredRuntimes();
    if (storedRuntimes.length === 0) {
      return defaultRuntimeSettings;
    }

    return storedRuntimes.map(runtime => ({
      alias: runtime.alias,
      url: runtime.url,
    }));
  }

  private status$ = this.store.select<RuntimeViewStatus>(
    store => store.runtimesLoadStatus
  );

  private runtimes$: Observable<Runtime[]> = this.store
    .select<Runtime[]>(store => store.runtimes)
    .pipe(
      withLatestFrom(this.status$),
      map(([runtimes, status]) => {
        if (status === RuntimeViewStatus.done) {
          window.localStorage.setItem('runtimes', JSON.stringify(runtimes));
        }

        return runtimes;
      })
    );

  getStatus() {
    return this.status$;
  }

  getRuntimes() {
    return this.runtimes$;
  }

  getRuntimeByAlias(alias: string) {
    return this.runtimes$.pipe(
      map(runtimes => runtimes.find(runtime => runtime.alias === alias)),
      map(runtime => {
        if (!runtime) {
          throw new Error(`No runtime of alias ${alias}`);
        }
        return runtime;
      }),
      shareReplay(1)
    );
  }

  registerRuntime(alias: string, url: string) {
    return of(null).pipe(
      withLatestFrom(this.runtimes$),
      tap(() => this.store.dispatch({type: 'runtime-register-start'})),
      map(([_, runtimes]) => {
        if (runtimes.some(runtime => runtime.alias === alias)) {
          throw Error(`Cannot have runtime of duplicated alias: ${alias}`);
        }
      }),
      mergeMap(() => this.getRuntimeInfo(url, alias)),
      tap(runtime => {
        if (runtime.status === 'error') {
          throw new Error(`Cannot register runtime ${alias} (url: ${url})`);
        }
      }),
      tap(runtime =>
        this.store.dispatch({
          type: 'runtime-register-complete',
          runtime,
        })
      ),
      catchError(error => {
        this.store.dispatch({
          type: 'runtime-register-error',
        });

        throw error;
      })
    );
  }

  unregisterRuntime(alias: string) {
    this.store.dispatch({
      type: 'runtime-unregister',
      alias,
    });
  }

  refreshRuntimes() {
    this.store.dispatch({
      type: 'runtime-refresh-start',
    });

    const settings = this.getInitRuntimeSettings();

    merge(settings.map(({url, alias}) => this.getRuntimeInfo(url, alias)))
      .pipe(mergeAll())
      .subscribe({
        complete: () => {
          this.store.dispatch({type: 'runtime-load-complete'});
        },
        next: runtime => this.store.dispatch({type: 'runtime-load', runtime}),
      });
  }

  private getGroups(hostUrl: string) {
    return this.apiService.listGroups(hostUrl).pipe(map(({groups}) => groups));
  }

  private getHosts(
    runtimeUrl: string,
    zone: string,
    runtimeAlias: string
  ): Observable<Host[]> {
    return this.apiService.listHosts(runtimeUrl, zone).pipe(
      map(({items: hosts}) => hosts || []),
      mergeMap(hosts => {
        return forkJoin(
          hosts.map(host => {
            const hostUrl = `${runtimeUrl}/v1/zones/${zone}/hosts/${host.name}`;
            return this.getGroups(hostUrl).pipe(
              map(groups => ({
                name: host.name!,
                zone: zone,
                url: hostUrl,
                runtime: runtimeAlias,
                groups,
              }))
            );
          })
        ).pipe(defaultIfEmpty([]));
      })
    );
  }

  getRuntimeInfo(url: string, alias: string): Observable<Runtime> {
    return this.apiService.getRuntimeInfo(url).pipe(
      switchMap(info => {
        // TODO: handle local workstation depending on type
        return this.apiService.listZones(url).pipe(
          map(({items: zones}) => zones || []),
          map(zones => ({
            type: info.type,
            zones: zones.map(zone => zone.name),
          }))
        );
      }),
      switchMap(({type, zones}) => {
        return forkJoin(
          zones.map(zone => this.getHosts(url, zone, alias))
        ).pipe(
          defaultIfEmpty([]),
          map(hostLists => hostLists.flat()),
          map((hosts: Host[]) => ({
            alias,
            type,
            url,
            zones,
            hosts,
            status: RuntimeStatus.valid,
          }))
        );
      }),

      catchError(error => {
        console.error(`Error from getRuntimeInfo of: ${alias} (${url})`);
        console.error(error);
        return of({
          alias,
          url,
          hosts: [],
          status: RuntimeStatus.error,
        });
      })
    );
  }

  constructor(
    private apiService: ApiService,
    private store: Store
  ) {
    this.refreshRuntimes();
  }
}
