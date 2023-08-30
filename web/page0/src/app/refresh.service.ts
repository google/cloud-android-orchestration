import {Injectable} from '@angular/core';
import {Store} from 'src/app/store/store';
import {defaultRuntimeSettings} from './settings';
import {forkJoin, Observable, Subscription} from 'rxjs';
import {defaultIfEmpty, map, switchMap, tap} from 'rxjs/operators';
import {Runtime} from 'src/app/interface/runtime-interface';
import {FetchService} from './fetch.service';

@Injectable({
  providedIn: 'root',
})
export class RefreshService {
  private prevSubscription: Subscription | undefined = undefined;

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

  refreshRuntime(url: string, alias: string): Observable<void> {
    return this.fetchService.fetchRuntime(url, alias).pipe(
      tap((runtime: Runtime) => {
        this.store.dispatch({
          type: 'runtime-load',
          runtime,
        });
      }),
      switchMap(runtime => this.fetchService.loadHosts(runtime)),
      map(() => {
        return;
      })
    );
  }

  refresh() {
    const settings = this.getInitRuntimeSettings();

    if (this.prevSubscription) {
      this.prevSubscription.unsubscribe();
    }

    const subscription = forkJoin(
      settings.map(({url, alias}) => this.refreshRuntime(url, alias))
    )
      .pipe(defaultIfEmpty([]))
      .subscribe({
        complete: () => {
          this.store.dispatch({type: 'refresh-complete'});
        },
      });

    this.store.dispatch({
      type: 'refresh-start',
    });

    this.prevSubscription = subscription;
  }

  constructor(
    private store: Store,
    private fetchService: FetchService
  ) {}
}
