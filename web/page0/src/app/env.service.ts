import {Injectable} from '@angular/core';
import {throwError} from 'rxjs';
import {catchError, tap} from 'rxjs/operators';
import {Store} from 'src/app/store/store';
import {ApiService} from './api.service';
import {GroupForm} from './interface/device-interface';
import {Environment, EnvStatus} from './interface/env-interface';
import {Group} from './interface/host-orchestrator.dto';
import {OperationService} from './operation.service';
import {hasDuplicate} from './utils';
@Injectable({
  providedIn: 'root',
})
export class EnvService {
  createEnv(runtimeAlias: string, hostUrl: string, groupForm: GroupForm) {
    // TODO: long polling

    const {groupName, devices} = groupForm;

    if (hasDuplicate(devices.map(device => device.deviceId))) {
      throw new Error('Devices in a group should have distinct ids');
    }

    return this.apiService
      .createGroup(hostUrl, {
        group_name: groupName,
        instance_names: devices.map(device => device.deviceId),
        cvd: {
          name: devices[0].deviceId,
          build_source: {
            android_ci_build_source: {
              main_build: {
                branch: devices[0].branch,
                build_id: devices[0].buildId,
                target: devices[0].target,
              },
            },
          },
          status: '',
          displays: [],
        },
      })
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
                groupName,
                runtimeAlias,
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
                    buildId: 'unknown',
                    target: 'unknown',
                    branch: 'unknown',
                    status: 'Running',
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

  deleteEnv(target: Environment) {}

  constructor(
    private apiService: ApiService,
    private store: Store,
    private waitService: OperationService
  ) {}
}
