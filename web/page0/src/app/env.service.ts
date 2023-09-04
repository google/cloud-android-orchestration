import {Injectable} from '@angular/core';
import {throwError} from 'rxjs';
import {catchError, switchMap, take, tap} from 'rxjs/operators';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {EnvStatus} from './interface/env-interface';
import {Group} from './interface/host-orchestrator.dto';
import {parseEnvConfig} from './interface/utils';
import jsonutils from './json.utils';
import {OperationService} from './operation.service';
import {hostSelectorFactory} from './store/selectors';
import {hasDuplicate} from './utils';
@Injectable({
  providedIn: 'root',
})
export class EnvService {
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

  constructor(
    private apiService: ApiService,
    private store: Store,
    private waitService: OperationService
  ) {}
}
