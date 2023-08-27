import {Injectable} from '@angular/core';
import {Observable, of} from 'rxjs';
import {
  map,
  shareReplay,
  tap,
  withLatestFrom,
  catchError,
  mergeMap,
} from 'rxjs/operators';
import {runtimeListSelector} from 'src/store/selectors';
import {Store} from 'src/store/store';
import {FetchService} from './fetch.service';
import {Runtime, RuntimeViewStatus} from './runtime-interface';

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private status$ = this.store.select<RuntimeViewStatus>(
    store => store.runtimesLoadStatus
  );

  private runtimes$: Observable<Runtime[]> = this.store
    .select<Runtime[]>(runtimeListSelector)
    .pipe(
      withLatestFrom(this.status$),
      map(([runtimes, status]) => {
        if (status === RuntimeViewStatus.done) {
          window.localStorage.setItem('runtimes', JSON.stringify(runtimes));
        }

        return runtimes;
      })
    );

  getRuntimes() {
    return this.runtimes$;
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
      mergeMap(() => this.fetchService.fetchRuntimeInfo(url, alias)),
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

  constructor(
    private store: Store,
    private fetchService: FetchService
  ) {}
}
