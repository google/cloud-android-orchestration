import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, scan, shareReplay, startWith, Subject, tap } from 'rxjs';
import { Runtime } from './runtime-interface';

interface RuntimeAction {
  type: 'register' | 'unregister';
  value: Runtime;
}

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
    scan((acc, action) => {
      if (action.type === 'register') {
        return [...acc, action.value];
      }
      return acc.filter((item) => item.alias !== action.value.alias);
    }, Array<Runtime>()),
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

  unregisterRuntime(runtime: Runtime) {
    this.runtimeAction.next({
      type: 'unregister',
      value: runtime,
    });
  }

  verifyRuntime(url: string): Observable<Runtime> {
    return this.httpClient.get<Runtime>(`${url}/verify`);
  }

  constructor(private httpClient: HttpClient) {}
}
