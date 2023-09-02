import {Injectable} from '@angular/core';
import {throwError} from 'rxjs';
import {catchError, tap} from 'rxjs/operators';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {EnvStatus} from './interface/env-interface';
import {Group} from './interface/host-orchestrator.dto';
import {parseEnvConfig} from './interface/utils';
import {OperationService} from './operation.service';
import {hasDuplicate} from './utils';
@Injectable({
  providedIn: 'root',
})
export class EnvService {
  createEnv(
    runtimeAlias: string | null | undefined,
    hostUrl: string | null | undefined,
    canonicalConfig: string | null | undefined
  ) {
    if (!runtimeAlias || !hostUrl || !canonicalConfig) {
      throw new Error('The form is not filled');
    }

    const {groupName, devices} = parseEnvConfig(canonicalConfig);

    if (hasDuplicate(devices.map(device => device.deviceId))) {
      throw new Error('Devices in a group should have distinct ids');
    }

    return this.apiService
      .createGroup(hostUrl, JSON.parse(canonicalConfig))
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
                  groupName, // TODO: The result should return real build id, target, branch
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
              throw error;
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
  }

  constructor(
    private apiService: ApiService,
    private store: Store,
    private waitService: OperationService
  ) {}
}
