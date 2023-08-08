import { Injectable } from '@angular/core';
import { map, mergeScan, of, shareReplay, startWith, Subject, tap } from 'rxjs';
import { ApiService } from './api.service';
import { HostInstance } from './cloud-orchestrator.dto';
import { Host } from './host-interface';
import { Runtime } from './runtime-interface';
import { RuntimeService } from './runtime.service';

interface HostCreateAction {
  type: 'create';
  host: Host;
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

  createHost(hostInstance: HostInstance, runtime: Runtime, zone: string) {
    // TODO: long polling
    return this.apiService.createHost(runtime.url, zone, {
      host_instance: hostInstance,
    });
  }

  deleteHost(hostUrl: string) {
    // TODO: long polling
    return this.apiService.deleteHost(hostUrl);
  }

  private hosts$ = this.hostAction.pipe(
    startWith<HostAction>({ type: 'init' }),
    tap((action) => console.log('host: ', action)),
    mergeScan((acc: Host[], action) => {
      if (action.type === 'init') {
        return this.runtimeService
          .getRuntimes()
          .pipe(
            map((runtimes) => runtimes.flatMap((runtime) => runtime.hosts))
          );
      }

      if (action.type === 'create') {
        return of([...acc, action.host]);
      }

      if (action.type === 'delete') {
        return of(acc.filter((item) => item.url !== action.hostUrl));
      }

      return of(acc);
    }, []),
    shareReplay(1)
  );

  getHostsByZone(runtime: string, zone: string) {
    return this.hosts$.pipe(
      map((hosts) =>
        hosts.filter((host) => host.runtime === runtime && host.zone === zone)
      )
    );
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
    private apiService: ApiService,
    private runtimeService: RuntimeService
  ) {}
}
