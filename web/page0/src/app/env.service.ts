import {Injectable} from '@angular/core';
import {Observable, of, throwError, timer} from 'rxjs';
import {catchError, map, retry, switchMap, take} from 'rxjs/operators';
import {ActionType} from 'src/app/store/actions';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {HostInstance, Operation} from './interface/cloud-orchestrator.dto';
import {Environment} from './interface/env-interface';
import {HostStatus} from './interface/host-interface';
import {DeviceSetting} from './interface/device-interface';
import {Group} from './interface/host-orchestrator.dto';
import {Result, ResultType} from './interface/result-interface';
import {groupToEnv, parseEnvConfig} from './interface/utils';
import jsonutils from './json.utils';
import {OperationService} from './operation.service';
import {DEFAULT_HOST_SETTING} from './settings';
import {hostSelectorFactory, runtimeSelectorFactory} from './store/selectors';
import {AUTO_CREATE_HOST, hasDuplicate} from './utils';

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

      if (hostName === AUTO_CREATE_HOST) {
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

  private handleCreateEnvStatus(
    hostUrl: string,
    runtimeAlias: string,
    groupName:string,
    devices: DeviceSetting[],
    result: Result<Group>
  ) {
    switch (result.type) {
      case ResultType.waitStarted:
        this.store.dispatch({
          type: ActionType.EnvCreateStart,
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
        break;
      case ResultType.done:
        const env = groupToEnv(runtimeAlias, hostUrl, {
          ...result.data,
          name: groupName,
        });

        this.store.dispatch({
          type: ActionType.EnvCreateComplete,
          waitUrl: result.waitUrl,
          env,
        });

        return {
          type: ResultType.done as ResultType.done,
          data: env,
          waitUrl: result.waitUrl,
        };
      default:
        break;
    }

    return result;
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
              return this.handleCreateEnvStatus(
                hostUrl, runtimeAlias, groupName, devices, result);
            })
          );
        }),

        catchError(error => {
          this.store.dispatch({
            type: ActionType.EnvCreateError,
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
              host_instance: DEFAULT_HOST_SETTING,
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
                    type: ActionType.EnvAutoHostCreateStart,
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
                            type: ActionType.EnvAutoHostCreateComplete,
                            waitUrl: hostCreateWaitUrl,
                            host: {
                              name: hostInstance.name!,
                              zone,
                              url: hostUrl,
                              runtime: runtime.alias,
                              status: HostStatus.running,
                            },
                          });
                        }

                        return this.handleCreateEnvStatus(
                          hostUrl, runtimeAlias, groupName, devices, result);
                      }),
                      catchError(error => {
                        this.store.dispatch({
                          type: ActionType.EnvCreateError,
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
            type: ActionType.HostCreateError,
          });

          return throwError(() => error);
        }),

        map(result => result!)
      );
  }
}
