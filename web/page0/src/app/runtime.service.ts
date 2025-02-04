import {Injectable, inject} from '@angular/core';
import {Observable, of} from 'rxjs';
import {map, tap, withLatestFrom, catchError, switchMap} from 'rxjs/operators';
import {
  runtimeListSelector,
  runtimesLoadStatusSelector,
} from 'src/app/store/selectors';
import {ActionType} from 'src/app/store/actions';
import {Store} from 'src/app/store/store';
import {FetchService} from './fetch.service';
import {Runtime, RuntimeViewStatus} from 'src/app/interface/runtime-interface';

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private store = inject(Store);
  private fetchService = inject(FetchService);
  private status$ = this.store.select(runtimesLoadStatusSelector);
  private runtimes$: Observable<Runtime[]> = this.store
  .select<Runtime[]>(runtimeListSelector)
  .pipe(withLatestFrom(this.status$), map(([runtimes, status]) => {
    if (status === RuntimeViewStatus.done) {
      window.localStorage.setItem('runtimes', JSON.stringify(runtimes));
    }
    return runtimes;
  }));

  getRuntimes() {
    return this.runtimes$;
  }

  registerRuntime(alias: string, url: string) {
    return of(null).pipe(
      withLatestFrom(this.runtimes$),
      tap(() => this.store.dispatch({type: ActionType.RuntimeRegisterStart})),
      map(([_, runtimes]) => {
        if (runtimes.some(runtime => runtime.alias === alias)) {
          throw Error(`Cannot have runtime of duplicated alias: ${alias}`);
        }
      }),
      switchMap(() => this.fetchService.fetchRuntime(url, alias)),
      tap(runtime => {
        if (runtime.status === 'error') {
          throw new Error(`Cannot register runtime ${alias} (url: ${url})`);
        }
      }),
      tap(runtime =>
        this.store.dispatch({
          type: ActionType.RuntimeRegisterComplete,
          runtime,
        })
      ),
      catchError(error => {
        this.store.dispatch({
          type: ActionType.RuntimeRegisterError,
        });

        throw error;
      })
    );
  }

  unregisterRuntime(alias: string) {
    this.store.dispatch({
      type: ActionType.RuntimeUnregister,
      alias,
    });
  }
}
