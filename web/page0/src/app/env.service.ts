import {Injectable} from '@angular/core';
import {Observable, of, Subject} from 'rxjs';
import {
  map,
  mergeScan,
  tap,
  shareReplay,
  scan,
  mergeWith,
} from 'rxjs/operators';
import {runtimeListSelector} from 'src/store/selectors';
import {Store} from 'src/store/store';
import {ApiService} from './api.service';
import {DeviceSetting, GroupForm} from './device-interface';
import {Environment, EnvStatus} from './env-interface';
import {Host} from './host-interface';
import {CVD} from './host-orchestrator.dto';
import {Runtime} from './runtime-interface';
import {hasDuplicate} from './utils';

interface EnvCreateAction {
  type: 'create';
  env: Environment;
}

interface EnvDeleteAction {
  type: 'delete';
  target: Environment;
}

interface EnvInitAction {
  type: 'init';
  envs: Environment[];
}

type EnvAction = EnvCreateAction | EnvDeleteAction | EnvInitAction;

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

  private cvdToDevice(cvd: CVD): DeviceSetting {
    const {name, build_source} = cvd;
    const {android_ci_build_source} = build_source;
    const {main_build} = android_ci_build_source;
    const {branch, build_id, target} = main_build;

    return {
      deviceId: name,
      branch,
      buildId: build_id,
      target,
    };
  }

  private hostToEnvList(host: Host): Environment[] {
    return host.groups.flatMap(group => ({
      runtimeAlias: host.runtime,
      hostUrl: host.url,
      groupName: group.name,
      devices: group.cvds.map(cvd => this.cvdToDevice(cvd)),
      status: EnvStatus.running,
    }));
  }

  private runtimeToEnvList(runtime: Runtime): Environment[] {
    return runtime.hosts.flatMap(host => this.hostToEnvList(host));
  }

  private envsFromRuntimes$: Observable<EnvInitAction> = this.store
    .select(runtimeListSelector)
    .pipe(
      map(runtimes =>
        runtimes.flatMap(runtime => this.runtimeToEnvList(runtime))
      ),
      map(envs => ({
        type: 'init',
        envs,
      }))
    );

  private isSame(env1: Environment, env2: Environment) {
    return env1.groupName === env2.groupName && env1.hostUrl === env2.hostUrl;
  }

  private environments: Observable<Environment[]> = this.envsFromRuntimes$.pipe(
    mergeWith(this.envAction),
    mergeScan((envs: Environment[], action) => {
      if (action.type === 'init') {
        return of(action.envs);
      }

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

  getEnvs() {
    return this.environments;
  }

  constructor(
    private apiService: ApiService,
    private store: Store
  ) {}
}
