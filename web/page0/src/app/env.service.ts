import {Injectable} from '@angular/core';
import {throwError} from 'rxjs';
import {catchError, map, retry, switchMap, take, tap} from 'rxjs/operators';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {HostInstance} from './interface/cloud-orchestrator.dto';
import {EnvStatus} from './interface/env-interface';
import {HostStatus} from './interface/host-interface';
import {Group} from './interface/host-orchestrator.dto';
import {parseEnvConfig} from './interface/utils';
import jsonutils from './json.utils';
import {OperationService} from './operation.service';
import {defaultHostSetting} from './settings';
import {hostSelectorFactory, runtimeSelectorFactory} from './store/selectors';
import {auto_create_host, hasDuplicate} from './utils';
@Injectable({
  providedIn: 'root',
})
export class EnvService {
  constructor(
    private apiService: ApiService,
    private store: Store,
    private waitService: OperationService
  ) {}

  createEnv(
    runtimeAlias: string | null | undefined,
    zone: string | null | undefined,
    hostName: string | null | undefined,
    canonicalConfig: string | null | undefined
  ) {
    if (!runtimeAlias || !zone || !hostName || !canonicalConfig) {
      return throwError(() => new Error('The form is not filled'));
    }

    const {groupName, devices} = parseEnvConfig(canonicalConfig);

    if (hasDuplicate(devices.map(device => device.deviceId))) {
      return throwError(
        () => new Error('Devices in a group should have distinct ids')
      );
    }

    if (hostName === auto_create_host) {
      return this.autoCreate(runtimeAlias, zone, canonicalConfig);
    }

    return this.store
      .select(hostSelectorFactory({runtimeAlias, zone, name: hostName}))
      .pipe(
        take(1),
        switchMap(host => {
          if (!host || !host.url) {
            return throwError(
              () =>
                new Error(
                  `Invalid host: ${hostName} (zone: ${zone}) does not exist in runtime ${runtimeAlias} `
                )
            );
          }

          const hostUrl = host.url!;
          return this.apiService
            .createGroup(hostUrl, jsonutils.parse(canonicalConfig))
            .pipe(
              tap(operation => {
                const waitUrl = `${hostUrl}/operations/${operation.name}`;

                this.store.dispatch({
                  type: 'env-create-start',
                  wait: {
                    waitUrl,
                    metadata: {
                      type: 'env-create',
                      hostUrl,
                      runtimeAlias,
                      groupName,
                      devices,
                    },
                  },
                });

                this.waitService.longPolling<Group>(waitUrl).subscribe({
                  next: (group: Group) => {
                    this.store.dispatch({
                      type: 'env-create-complete',
                      waitUrl,
                      env: {
                        runtimeAlias,
                        hostUrl,
                        groupName: groupName || 'unknown', // TODO: The result should return real build id, target, branch
                        devices: group.cvds.map(cvd => ({
                          deviceId: cvd.name,
                          branch_or_buildId: 'unknown',
                          target: 'unknown',
                          status: cvd.status,
                          displays: cvd.displays,
                        })),
                        status: EnvStatus.running,
                      },
                    });
                  },
                  error: error => {
                    this.store.dispatch({
                      type: 'env-create-error',
                      waitUrl,
                    });
                    return throwError(() => error);
                  },
                });
              }),
              catchError(error => {
                this.store.dispatch({
                  type: 'env-create-error',
                });
                return throwError(() => error);
              })
            );
        })
      );
  }

  autoCreate(runtimeAlias: string, zone: string, canonicalConfig: string) {
    const {groupName, devices} = parseEnvConfig(canonicalConfig);

    return this.store
      .select(runtimeSelectorFactory({alias: runtimeAlias}))
      .pipe(
        take(1),
        map(runtime => {
          if (!runtime) {
            throw new Error(`Invalid runtime: ${runtimeAlias}`);
          }

          return runtime;
        }),

        switchMap(runtime =>
          this.apiService
            .createHost(runtime.url, zone, {
              host_instance: defaultHostSetting,
            })
            .pipe(
              tap(operation => {
                const hostCreateWaitUrl = `${runtime.url}/v1/zones/${zone}/operations/${operation.name}`;

                this.store.dispatch({
                  type: 'env-auto-host-create-start',
                  wait: {
                    waitUrl: hostCreateWaitUrl,
                    metadata: {
                      type: 'env-auto-host-create',
                      zone,
                      runtimeAlias: runtime.alias,
                      groupName,
                      devices,
                    },
                  },
                });

                this.waitService
                  .longPolling<HostInstance>(hostCreateWaitUrl)
                  .pipe(
                    switchMap(hostInstance => {
                      const hostUrl = `${runtime.url}/v1/zones/${zone}/hosts/${hostInstance.name}`;

                      return this.apiService
                        .createGroup(hostUrl, jsonutils.parse(canonicalConfig))
                        .pipe(
                          retry({
                            delay: 1000,
                          }),
                          tap(operation => {
                            this.store.dispatch({
                              type: 'env-auto-host-create-complete',
                              waitUrl: hostCreateWaitUrl,
                              host: {
                                name: hostInstance.name!,
                                zone,
                                url: hostUrl,
                                runtime: runtime.alias,
                                status: HostStatus.running,
                              },
                            });

                            const envCreateWaitUrl = `${hostUrl}/operations/${operation.name}`;

                            this.store.dispatch({
                              type: 'env-create-start',
                              wait: {
                                waitUrl: envCreateWaitUrl,
                                metadata: {
                                  type: 'env-create',
                                  hostUrl,
                                  runtimeAlias: runtime.alias,
                                  groupName,
                                  devices,
                                },
                              },
                            });

                            this.waitService
                              .longPolling<Group>(envCreateWaitUrl)
                              .subscribe({
                                next: (group: Group) => {
                                  this.store.dispatch({
                                    type: 'env-create-complete',
                                    waitUrl: envCreateWaitUrl,
                                    env: {
                                      runtimeAlias: runtime.alias,
                                      hostUrl,
                                      groupName: groupName || 'unknown', // TODO: The result should return real build id, target, branch
                                      devices: group.cvds.map(cvd => ({
                                        deviceId: cvd.name,
                                        branch_or_buildId: 'unknown',
                                        target: 'unknown',
                                        status: cvd.status,
                                        displays: cvd.displays,
                                      })),
                                      status: EnvStatus.running,
                                    },
                                  });
                                },
                                error: error => {
                                  this.store.dispatch({
                                    type: 'env-create-error',
                                    waitUrl: envCreateWaitUrl,
                                  });
                                  return throwError(() => error);
                                },
                              });
                          }),
                          catchError(error => {
                            this.store.dispatch({
                              type: 'env-create-error',
                            });
                            return throwError(() => error);
                          })
                        );
                    })
                  )
                  .subscribe({
                    error: error => {
                      this.store.dispatch({
                        type: 'host-create-error',
                        waitUrl: hostCreateWaitUrl,
                      });
                      throw error;
                    },
                  });
              })
            )
        ),

        catchError(error => {
          this.store.dispatch({
            type: 'host-create-error',
          });

          return throwError(() => error);
        })
      );
  }
}
