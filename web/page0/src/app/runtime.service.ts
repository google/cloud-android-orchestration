import { Injectable } from '@angular/core';
import {
  BehaviorSubject,
  catchError,
  forkJoin,
  map,
  mergeMap,
  mergeScan,
  Observable,
  of,
  shareReplay,
  startWith,
  Subject,
  switchMap,
  tap,
  withLatestFrom,
} from 'rxjs';
import { ApiService } from './api.service';
import { Host } from './host-interface';
import { Runtime, RuntimeViewStatus, RuntimeStatus } from './runtime-interface';
import { defaultRuntimeSettings } from './settings';

interface RuntimeRegisterAction {
  type: 'register';
  runtime: Runtime;
}

interface RuntimeUnregisterAction {
  type: 'unregister';
  alias: string;
}

interface RuntimeRefreshAction {
  type: 'refresh';
}

interface RuntimeInitializeAction {
  type: 'init';
}

type RuntimeAction =
  | RuntimeRegisterAction
  | RuntimeUnregisterAction
  | RuntimeRefreshAction
  | RuntimeInitializeAction;

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private runtimeAction = new Subject<RuntimeAction>();

  private register(runtime: Runtime) {
    this.runtimeAction.next({
      type: 'register',
      runtime,
    });
  }

  private unregister(alias: string) {
    this.runtimeAction.next({
      type: 'unregister',
      alias,
    });
  }

  private refresh() {
    this.runtimeAction.next({
      type: 'refresh',
    });
  }

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

    return storedRuntimes.map((runtime) => ({
      alias: runtime.alias,
      url: runtime.url,
    }));
  }

  private runtimes$: Observable<Runtime[]> = this.runtimeAction.pipe(
    startWith<RuntimeAction>({ type: 'init' }),
    tap((action) => console.log(action)),
    mergeScan((acc: Runtime[], action) => {
      if (action.type === 'init') {
        this.status$.next(RuntimeViewStatus.initializing);

        return forkJoin(
          this.getInitRuntimeSettings().map(({ alias, url }) =>
            this.getRuntimeInfo(url, alias)
          )
        ).pipe(
          tap((runtimes) => {
            this.status$.next(RuntimeViewStatus.done);
          })
        );
      }

      if (action.type === 'register') {
        return of([...acc, action.runtime]);
      }

      if (action.type === 'unregister') {
        return of(acc.filter((item) => item.alias !== action.alias));
      }

      if (action.type === 'refresh') {
        this.status$.next(RuntimeViewStatus.refreshing);
        return forkJoin(
          acc.map((runtime) => this.getRuntimeInfo(runtime.url, runtime.alias))
        ).pipe(
          tap(() => {
            this.status$.next(RuntimeViewStatus.done);
          })
        );
      }
      return of(acc);
    }, []),
    tap((runtimes) => console.log('runtimes', runtimes)),
    tap((runtimes) =>
      window.localStorage.setItem('runtimes', JSON.stringify(runtimes))
    ),
    shareReplay(1)
  );

  private status$ = new BehaviorSubject<RuntimeViewStatus>(
    RuntimeViewStatus.initializing
  );

  getStatus() {
    return this.status$;
  }

  getRuntimes() {
    return this.runtimes$;
  }

  getRuntimeByAlias(alias: string) {
    return this.runtimes$.pipe(
      map((runtimes) => runtimes.find((runtime) => runtime.alias === alias)),
      map((runtime) => {
        if (!runtime) {
          throw new Error(`No runtime of alias ${alias}`);
        }
        return runtime;
      }),
      shareReplay(1)
    );
  }

  registerRuntime(alias: string, url: string) {
    this.status$.next(RuntimeViewStatus.registering);

    return of(null).pipe(
      withLatestFrom(this.runtimes$),
      map(([_, runtimes]) => {
        if (runtimes.some((runtime) => runtime.alias === alias)) {
          throw Error(`Cannot have runtime of duplicated alias: ${alias}`);
        }
      }),
      mergeMap(() => this.getRuntimeInfo(url, alias)),
      tap((runtime) => {
        if (runtime.status === 'error') {
          throw new Error(`Cannot register runtime ${alias} (url: ${url}`);
        }
      }),
      tap((runtime) => this.register(runtime)),
      tap(() => this.status$.next(RuntimeViewStatus.done)),
      catchError((error) => {
        this.status$.next(RuntimeViewStatus.register_error);
        throw error;
      })
    );
  }

  unregisterRuntime(alias: string) {
    this.unregister(alias);
  }

  refreshRuntimes() {
    this.refresh();
  }

  private getGroups(hostUrl: string) {
    return this.apiService
      .listGroups(hostUrl)
      .pipe(map(({ groups }) => groups));
  }

  private getHosts(
    runtimeUrl: string,
    zone: string,
    runtimeAlias: string
  ): Observable<Host[]> {
    return this.apiService.listHosts(runtimeUrl, zone).pipe(
      mergeMap(({ items: hosts }) => {
        return forkJoin(
          hosts.map((host) => {
            const hostUrl = `${runtimeUrl}/v1/zones/${zone}/hosts/${host.name}`;
            return this.getGroups(hostUrl).pipe(
              map((groups) => ({
                name: host.name!,
                zone: zone,
                url: hostUrl,
                runtime: runtimeAlias,
                groups,
              }))
            );
          })
        );
      })
    );
  }

  private getRuntimeInfo(url: string, alias: string): Observable<Runtime> {
    return this.apiService.getRuntimeInfo(url).pipe(
      switchMap((info) => {
        // TODO: handle local workstation depending on type
        return this.apiService.listZones(url).pipe(
          map(({ items: zones }) => ({
            type: info.type,
            zones,
          }))
        );
      }),

      switchMap(({ type, zones }) => {
        return forkJoin(
          zones.map((zone) => this.getHosts(url, zone, alias))
        ).pipe(
          map((hostLists) => hostLists.flat()),
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

      catchError((error) => {
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

  constructor(private apiService: ApiService) {}
}
