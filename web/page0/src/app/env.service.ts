import {Injectable} from '@angular/core';
import {Observable, of, Subject} from 'rxjs';
import {mergeScan, tap, shareReplay, scan} from 'rxjs/operators';
import {Store} from 'src/store/store';
import {ApiService} from './api.service';
import {GroupForm} from './device-interface';
import {Environment, EnvStatus} from './env-interface';
import {hasDuplicate} from './utils';

interface EnvCreateAction {
  type: 'create';
  env: Environment;
}

interface EnvDeleteAction {
  type: 'delete';
  target: Environment;
}

type EnvAction = EnvCreateAction | EnvDeleteAction;

@Injectable({
  providedIn: 'root',
})
export class EnvService {
  private envAction = new Subject<EnvAction>();

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
          this.envAction.next({
            type: 'create',
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

  private environments: Observable<Environment[]> = this.envAction.pipe(
    mergeScan((envs: Environment[], action) => {
      if (action.type === 'delete') {
        return of(
          envs.map(env => {
            if (this.isSame(env, action.target)) {
              env.status = EnvStatus.stopping;
            }
            return env;
          })
        );
      }

      if (action.type === 'create') {
        return of([...envs, action.env]);
      }

      return of(envs);
    }, []),
    scan((oldEnvs, newEnvs) => {
      const oldStarting = oldEnvs.filter(
        env => env.status === EnvStatus.starting
      );

      const starting = oldStarting.filter(
        oldEnv => !newEnvs.find(newEnv => this.isSame(oldEnv, newEnv))
      );

      const oldStopping = oldEnvs.filter(
        env => env.status === EnvStatus.stopping
      );

      const stoppingAndRunning = newEnvs.map(newEnv => {
        if (oldStopping.find(oldEnv => this.isSame(oldEnv, newEnv))) {
          newEnv.status = EnvStatus.stopping;
        }
        return newEnv;
      });

      return [...starting, ...stoppingAndRunning];
    }),
    shareReplay(1)
  );

  constructor(
    private apiService: ApiService,
    private store: Store
  ) {}
}
