import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
  catchError,
  first,
  forkJoin,
  map,
  mergeScan,
  Observable,
  of,
  shareReplay,
  startWith,
  Subject,
  tap,
} from 'rxjs';
import { Runtime, RuntimeAdditionalInfo } from './runtime-interface';
import { defaultRuntimeSettings } from './settings';

interface RuntimeRegisterAction {
  type: 'register';
  value: Runtime;
}

interface RuntimeUnregisterAction {
  type: 'unregister';
  value: string;
}

interface RuntimeRefreshAction {
  type: 'refresh';
  loading$: Subject<boolean>;
}

type RuntimeAction =
  | RuntimeRegisterAction
  | RuntimeUnregisterAction
  | RuntimeRefreshAction;

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private getStoredRuntimes(): Runtime[] {
    const runtimes = localStorage.getItem('runtimes');
    // TODO: handle type error
    if (runtimes) {
      return JSON.parse(runtimes) as Runtime[];
    }

    return [];
  }

  private initialized = false;

  private runtimeAction = new Subject<RuntimeAction>();

  private storedRuntimes: Runtime[] = this.getStoredRuntimes();

  private runtimes$: Observable<Runtime[]> = this.runtimeAction.pipe(
    tap((action) => console.log(action)),
    mergeScan((acc, action) => {
      if (action.type === 'register') {
        return of([...acc, action.value]);
      }

      if (action.type === 'unregister') {
        return of(acc.filter((item) => item.alias !== action.value));
      }

      if (action.type === 'refresh') {
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
            action.loading$.next(false);
          })
        );
      }

      return of(acc);
    }, this.storedRuntimes),
    startWith(this.storedRuntimes),
    tap((runtimes) => console.log('runtimes', runtimes)),
    tap((runtimes) =>
      localStorage.setItem('runtimes', JSON.stringify(runtimes))
    ),
    shareReplay(1)
  );

  getRuntimes() {
    return this.runtimes$;
  }

  registerRuntime(runtime: Runtime) {
    this.runtimeAction.next({
      type: 'register',
      value: runtime,
    });
  }

  unregisterRuntime(alias: string) {
    this.runtimeAction.next({
      type: 'unregister',
      value: alias,
    });
  }

  checkDuplicatedAlias(alias: string): Observable<void> {
    return this.runtimes$.pipe(
      map((runtimes) => {
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

  refreshRuntimes(loading$: Subject<boolean>) {
    loading$.next(true);
    this.runtimeAction.next({
      type: 'refresh',
      loading$,
    });
  }

  initRuntimes(loading$: Subject<boolean>) {
    if (this.initialized) {
      return;
    }

    this.initialized = true;

    const storedRuntimes = this.getStoredRuntimes();
    const toRegisterSettings = defaultRuntimeSettings.filter(
      ({ alias }) => !storedRuntimes.some((runtime) => runtime.alias === alias)
    );

    if (toRegisterSettings.length === 0) {
      return;
    }

    loading$.next(true);

    forkJoin(
      toRegisterSettings.map(({ alias, url }) =>
        this.verifyRuntime(url, alias).pipe(
          map((runtime) => {
            this.registerRuntime(runtime);
            return of({ success: true });
          }),
          catchError((error) => {
            console.error(`Failed to register runtime: ${alias} (${url})`);
            console.error(error);
            return of({ success: false });
          })
        )
      )
    )
      .pipe(first())
      .subscribe((res) => {
        loading$.next(false);
      });
  }

  constructor(private httpClient: HttpClient) {}
}
