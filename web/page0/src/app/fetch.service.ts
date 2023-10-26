import {Injectable} from '@angular/core';
import {forkJoin, merge, Observable, of} from 'rxjs';
import {
  map,
  catchError,
  defaultIfEmpty,
  switchMap,
  tap,
  retry,
  mergeAll,
  mergeMap,
} from 'rxjs/operators';
import {ApiService} from './api.service';
import {Host, HostStatus} from 'src/app/interface/host-interface';
import {Group} from 'src/app/interface/host-orchestrator.dto';
import {Runtime, RuntimeStatus} from 'src/app/interface/runtime-interface';
import {ActionType} from 'src/app/store/actions';
import {Store} from './store/store';
import {Environment, EnvStatus} from './interface/env-interface';
import {configToInfo, cvdToDevice} from 'src/app/interface/utils';

@Injectable({
  providedIn: 'root',
})
export class FetchService {
  fetchEnvs(runtimeAlias: string, hostUrl: string): Observable<Environment[]> {
    return this.apiService.listGroups(hostUrl).pipe(
      retry(1000),
      switchMap(groups => {
        return forkJoin(
          groups.map(group => {
            return this.apiService.listDevicesByGroup(hostUrl, group).pipe(
              map(device => {
                const cvds = device.map(device => ({
                  name: device.device_id,
                  build_source: {
                    android_ci_build_source: {
                      main_build: {
                        branch: '',
                        build_id: '',
                        target: '',
                      },
                    },
                  },
                  status: 'running',
                  displays: [],
                }));

                return {
                  name: group,
                  cvds,
                };
              })
            );
          })
        ).pipe(
          defaultIfEmpty([]),
          map((groups: Group[]) => {
            return groups.map(group => ({
              runtimeAlias,
              hostUrl,
              groupName: group.name,
              devices: group.cvds.map(cvd => cvdToDevice(cvd)),
              status: EnvStatus.running,
            }));
          })
        );
      })
    );
  }

  fetchHosts(
    runtimeUrl: string,
    zone: string,
    runtimeAlias: string
  ): Observable<Host[]> {
    return this.apiService.listHosts(runtimeUrl, zone).pipe(
      map(({items: hostInstances}) => hostInstances || []),
      map(hostInstances =>
        hostInstances.map(hostInstance => {
          const hostUrl = `${runtimeUrl}/v1/zones/${zone}/hosts/${hostInstance.name!}`;

          return {
            name: hostInstance.name!,
            runtime: runtimeAlias,
            zone: zone,
            url: hostUrl,
            status: HostStatus.running,
          };
        })
      )
    );
  }

  fetchRuntime(url: string, alias: string): Observable<Runtime> {
    return this.apiService.getRuntimeConfig(url).pipe(
      map(configToInfo),
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

      map(({type, zones}) => {
        return {
          alias,
          type,
          url,
          zones,
          status: RuntimeStatus.valid,
        };
      }),

      catchError(error => {
        console.error(`Error from fetchRuntime of: ${alias} (${url})`);
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

  loadHosts(runtime: Runtime): Observable<any> {
    if (runtime.status !== RuntimeStatus.valid || !runtime.zones) {
      return of([]);
    }

    const dispatchHostsLoad = (hosts: Host[]) => {
      hosts.forEach(host => {
        this.store.dispatch({
          type: ActionType.HostLoad,
          host,
        });
      });
    };

    const dispatchEnvsLoad = (envs: Environment[]) => {
      envs.forEach(env => {
        this.store.dispatch({
          type: ActionType.EnvLoad,
          env,
        });
      });
    };

    const hostList$ = merge(
      runtime.zones.map(zone =>
        this.fetchHosts(runtime.url, zone, runtime.alias)
      )
    ).pipe(mergeAll());

    const envs$ = hostList$.pipe(
      mergeMap(hosts => {
        return merge(
          hosts.flatMap(host => this.fetchEnvs(host.runtime, host.url!))
        ).pipe(mergeAll());
      })
    );

    return forkJoin([
      hostList$.pipe(tap(dispatchHostsLoad)),
      envs$.pipe(tap(dispatchEnvsLoad)),
    ]).pipe(defaultIfEmpty([]));
  }

  constructor(
    private apiService: ApiService,
    private store: Store
  ) {}
}
