import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
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
}

type RuntimeAction =
  | RuntimeRegisterAction
  | RuntimeUnregisterAction
  | RuntimeRefreshAction;

@Injectable({
  providedIn: 'root',
})
export class RuntimeService {
  private getDefaultRuntimesFInitiallStorage(): Runtime[] {
    const runtimes = localStorage.getItem('runtimes');
    // TODO: handle type error
    if (runtimes) {
      return JSON.parse(runtimes) as Runtime[];
    }

    return [];
  }

  private getDefaultRuntimesInitialaultFile(): Runtime[] {
    // TODO: load from default.json
    return [];
  }

  private getInitialRuntimes() {
    // TODO: handle alias duplicate
    const storedRuntimes = this.getDefaultRuntimesFInitiallStorage();
    const defaultRuntimes = this.getDefaultRuntimesInitialaultFile();

    const runtimeSet = new Set([...storedRuntimes, ...defaultRuntimes]);
    return Array.from(runtimeSet);
  }

  private runtimeAction = new Subject<RuntimeAction>();

  private initialRuntimes: Runtime[] = this.getInitialRuntimes();

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
          acc.map((runtime) => this.verifyRuntime(runtime.url, runtime.alias))
        );
      }
      return of(acc);
    }, this.initialRuntimes),
    startWith(this.initialRuntimes),
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

  verifyRuntime(url: string, alias: string): Observable<Runtime> {
    // TODO: check duplicate alias here
    return this.httpClient.get<RuntimeAdditionalInfo>(`${url}/verify`).pipe(
      map((info) => ({
        ...info,
        url,
        alias,
      }))
    );
  }

  refreshRuntimes() {
    this.runtimeAction.next({
      type: 'refresh',
    });
  }

  constructor(private httpClient: HttpClient) {}
}
