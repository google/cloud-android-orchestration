import {Injectable} from '@angular/core';
import {Observable, of, throwError, timer} from 'rxjs';
import {catchError, map, retry, switchMap, take} from 'rxjs/operators';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {HostInstance, Operation} from './interface/cloud-orchestrator.dto';
import {Environment} from './interface/env-interface';
import {HostStatus} from './interface/host-interface';
import {Group} from './interface/host-orchestrator.dto';
import {Result, ResultType} from './interface/result-interface';
import {groupToEnv, parseEnvConfig} from './interface/utils';
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
  ): Observable<Result<Environment>> {
    try {
      this.validateInput(runtimeAlias, zone, hostName, canonicalConfig);

      if (hostName === auto_create_host) {
        return this.createEnvInAutoHost(runtimeAlias!, zone!, canonicalConfig!);
      }

      return this.createEnvInSelectedHost(
        runtimeAlias!,
        zone!,
        hostName!,
        canonicalConfig!
      );
    } catch (error) {
      return throwError(() => error);
    }
  }

  private validateInput(
    runtimeAlias: string | null | undefined,
    zone: string | null | undefined,
    hostName: string | null | undefined,
    canonicalConfig: string | null | undefined
  ) {
    if (!runtimeAlias || !zone || !hostName || !canonicalConfig) {
      throw new Error('The form is not filled');
    }

    const {groupName, devices} = parseEnvConfig(canonicalConfig);

    // TODO: groupName cannot have dash(-) in it

    if (hasDuplicate(devices.map(device => device.deviceId))) {
      throw new Error('Devices in a group should have distinct ids');
    }
  }

  private createEnvInSelectedHost(
    runtimeAlias: string,
    zone: string,
    hostName: string,
    canonicalConfig: string
  ): Observable<Result<Environment>> {
    return this.store
      .select(hostSelectorFactory({runtimeAlias, zone, name: hostName}))
      .pipe(
        take(1),
        map(host => {
          if (!host || !host.url) {
            throw new Error(
              `Invalid host: ${hostName} (zone: ${zone}) does not exist in runtime ${runtimeAlias} `
            );
          }

          return host;
        }),
        switchMap(host => {
          const hostUrl = host.url!;
          const {groupName, devices} = parseEnvConfig(canonicalConfig);

          const request = this.apiService.createGroup(
            hostUrl,
            jsonutils.parse(canonicalConfig)
          );

          const waitUrlSynthesizer = (operation: Operation) =>
            `${hostUrl}/operations/${operation.name}`;

          return this.waitService.wait<Group>(request, waitUrlSynthesizer).pipe(
            map(result => {
              if (result.type === ResultType.waitStarted) {
                this.store.dispatch({
                  type: 'env-create-start',
                  wait: {
                    waitUrl: result.waitUrl,
                    metadata: {
                      type: 'env-create',
                      hostUrl,
                      runtimeAlias,
                      groupName,
                      devices,
                    },
                  },
                });

                return result;
              }

              if (result.type === ResultType.done) {
                const env = groupToEnv(runtimeAlias, hostUrl, {
                  ...result.data,
                  name: groupName,
                });

                this.store.dispatch({
                  type: 'env-create-complete',
                  waitUrl: result.waitUrl,
                  env,
                });

                return {
                  type: ResultType.done as ResultType.done,
                  data: env,
                  waitUrl: result.waitUrl,
                };
              }

              return result;
            })
          );
        }),

        catchError(error => {
          this.store.dispatch({
            type: 'env-create-error',
          });
          return throwError(() => error);
        })
      );
  }

  private createEnvInAutoHost(
    runtimeAlias: string,
    zone: string,
    canonicalConfig: string
  ) {
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
        switchMap(runtime => {
          const hostCreateRequest = this.apiService.createHost(
            runtime.url,
            zone,
            {
              host_instance: defaultHostSetting,
            }
          );

          const waitUrlSynthesizer = (operation: Operation) =>
            `${runtime.url}/v1/zones/${zone}/operations/${operation.name}`;

          return this.waitService
            .wait<HostInstance>(hostCreateRequest, waitUrlSynthesizer)
            .pipe(
              switchMap((result: Result<HostInstance>) => {
                if (result.type === ResultType.waitStarted) {
                  this.store.dispatch({
                    type: 'env-auto-host-create-start',
                    wait: {
                      waitUrl: result.waitUrl,
                      metadata: {
                        type: 'env-auto-host-create',
                        zone,
                        runtimeAlias: runtime.alias,
                        groupName,
                        devices,
                      },
                    },
                  });

                  return of(result);
                }

                if (result.type === ResultType.done) {
                  const hostInstance = result.data;
                  const hostUrl = `${runtime.url}/v1/zones/${zone}/hosts/${hostInstance.name}`;
                  const hostCreateWaitUrl = result.waitUrl;

                  const groupCreateRetryConfig = {
                    count: 1000,
                    delay: (err: unknown, retryCount: number) => {
                      return timer(1000);
                    },
                  };

                  const groupCreateRequest = this.apiService
                    .createGroup(hostUrl, jsonutils.parse(canonicalConfig))
                    .pipe(retry(groupCreateRetryConfig));

                  const waitUrlSynthesizer = (op: Operation) => {
                    return `${hostUrl}/operations/${op.name}`;
                  };

                  return this.waitService
                    .wait<Group>(groupCreateRequest, waitUrlSynthesizer)
                    .pipe(
                      map((result: Result<Group>) => {
                        if (result.type === ResultType.waitStarted) {
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

                          this.store.dispatch({
                            type: 'env-create-start',
                            wait: {
                              waitUrl: result.waitUrl,
                              metadata: {
                                type: 'env-create',
                                hostUrl,
                                runtimeAlias: runtime.alias,
                                groupName,
                                devices,
                              },
                            },
                          });
                          return result;
                        }

                        if (result.type === ResultType.done) {
                          const env = groupToEnv(runtime.alias, hostUrl, {
                            ...result.data,
                            name: groupName,
                          });

                          this.store.dispatch({
                            type: 'env-create-complete',
                            waitUrl: result.waitUrl,
                            env,
                          });

                          return {
                            type: result.type,
                            waitUrl: result.waitUrl,
                            data: env,
                          };
                        }

                        return result;
                      }),
                      catchError(error => {
                        this.store.dispatch({
                          type: 'env-create-error',
                        });
                        return throwError(() => error);
                      })
                    );
                }

                return of(result);
              })
            );
        }),

        catchError(error => {
          this.store.dispatch({
            type: 'host-create-error',
          });

          return throwError(() => error);
        }),

        map(result => result!)
      );
  }
}
