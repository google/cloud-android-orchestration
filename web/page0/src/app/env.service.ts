import {Injectable} from '@angular/core';
import {Observable, of, Subject} from 'rxjs';
import {mergeScan, tap, shareReplay, scan} from 'rxjs/operators';
import {Store} from 'src/store/store';
import {ApiService} from './api.service';
import {GroupForm} from './device-interface';
import {Environment, EnvStatus} from './env-interface';
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
        tap(() => {
          this.store.dispatch({
            type: 'env-create-start',
            env: {
              runtimeAlias,
              hostUrl,
              groupName,
              devices,
              status: EnvStatus.starting,
            },
          });
        })
      );
  }

  private isSame(env1: Environment, env2: Environment) {
    return env1.groupName === env2.groupName && env1.hostUrl === env2.hostUrl;
  }

  deleteEnv(target: Environment) {
    // TODO: long polling
    this.store.dispatch({type: 'env-delete-start', target});
  }

  constructor(
    private apiService: ApiService,
    private store: Store
  ) {}
}
