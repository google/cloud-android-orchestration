import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
  BehaviorSubject,
  catchError,
  first,
  forkJoin,
  map,
  mergeMap,
  mergeScan,
  Observable,
  of,
  shareReplay,
  startWith,
  Subject,
  tap,
  withLatestFrom,
} from 'rxjs';
import {
  Runtime,
  RuntimeAdditionalInfo,
  RuntimesStatus,
} from './runtime-interface';
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
    const runtimes = localStorage.getItem('runtimes');
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

  private initRuntimeSettings = this.getInitRuntimeSettings();

  private runtimes$: Observable<Runtime[]> = this.runtimeAction.pipe(
    startWith({ type: 'init' } as RuntimeInitializeAction),
    tap((action) => console.log(action)),
    mergeScan((acc, action) => {
      if (action.type === 'init') {
        this.status$.next(RuntimesStatus.initializing);

        return forkJoin(
          this.initRuntimeSettings.map(({ alias, url }) =>
            this.verifyRuntime(url, alias).pipe(
              catchError((error) => {
                console.error(error);
                return of({
                  url,
                  alias,
                  status: 'error',
                } as Runtime);
              })
            )
          )
        ).pipe(
          tap((runtimes) => {
            this.status$.next(RuntimesStatus.done);
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
        this.status$.next(RuntimesStatus.refreshing);
        return forkJoin(
          acc.map((runtime) =>
            this.verifyRuntime(runtime.url, runtime.alias).pipe(
              catchError((error) =>
                of({
                  url: runtime.url,
                  alias: runtime.alias,
                  status: 'error',
                } as Runtime)
              )
            )
          )
        ).pipe(
          tap(() => {
            this.status$.next(RuntimesStatus.done);
          })
        );
      }
      return of(acc);
    }, [] as Runtime[]),
    tap((runtimes) => console.log('runtimes', runtimes)),
    tap((runtimes) =>
      localStorage.setItem('runtimes', JSON.stringify(runtimes))
    ),
    shareReplay(1)
  );

  private status$ = new BehaviorSubject<RuntimesStatus>(
    RuntimesStatus.initializing
  );

  getStatus() {
    return this.status$;
  }

  getRuntimes() {
    return this.runtimes$;
  }

  registerRuntime(alias: string, url: string) {
    this.status$.next(RuntimesStatus.registering);

    return this.checkDuplicatedAlias(alias).pipe(
      mergeMap(() => this.verifyRuntime(url, alias)),
      map((runtime) => this.register(runtime)),
      tap(() => this.status$.next(RuntimesStatus.done)),
      catchError((error) => {
        this.status$.next(RuntimesStatus.register_error);
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

  private checkDuplicatedAlias(alias: string): Observable<void> {
    return of(null).pipe(
      withLatestFrom(this.runtimes$),
      map(([_, runtimes]) => {
        if (runtimes.some((runtime) => runtime.alias === alias)) {
          throw Error(`Cannot have runtime of duplicated alias: ${alias}`);
        }
      })
    );
  }

  verifyRuntime(url: string, alias: string): Observable<Runtime> {
    return this.httpClient.get<RuntimeAdditionalInfo>(`${url}/verify`).pipe(
      map(
        (info) =>
          ({
            ...info,
            url,
            alias,
            status: 'valid',
          } as Runtime)
      ),
      first()
    );
  }

  constructor(private httpClient: HttpClient) {}
}
