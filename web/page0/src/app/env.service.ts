import { Injectable } from '@angular/core';
import {
  map,
  merge,
  mergeScan,
  Observable,
  of,
  scan,
  shareReplay,
  Subject,
  tap,
} from 'rxjs';
import { ApiService } from './api.service';
import { DeviceSetting } from './device-interface';
import { Environment, EnvStatus } from './env-interface';
import { Host } from './host-interface';
import { CVD } from './host-orchestrator.dto';
import { Runtime } from './runtime-interface';
import { RuntimeService } from './runtime.service';
import { hasDuplicate } from './utils';

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

  createEnv(
    runtimeAlias: string,
    hostUrl: string,
    groupForm: {
      groupName: string;
      devices: {
        deviceId: string;
        branch: string;
        target: string;
        buildId: string;
      }[];
    }
  ) {
    // TODO: long polling

    const { groupName, devices } = groupForm;

    if (hasDuplicate(devices.map((device) => device.deviceId))) {
      throw new Error('Devices in a group should have distinct ids');
    }

    return this.apiService
      .createGroup(hostUrl, {
        group: {
          name: groupName,
          cvds: devices.map((device) => ({
            name: device.deviceId,
            build_source: {
              android_ci_build_source: {
                main_build: {
                  branch: device.branch,
                  build_id: device.buildId,
                  target: device.target,
                },
              },
            },
            status: '',
            displays: [],
            group_name: groupName,
          })),
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

  deleteEnv(target: Environment) {
    // TODO: long polling
    const { hostUrl, groupName } = target;
    this.envAction.next({ type: 'delete', target });

    return this.apiService.deleteGroup(hostUrl, groupName).subscribe();
  }

  private cvdToDevice(cvd: CVD): DeviceSetting {
    const { name, build_source } = cvd;
    const { android_ci_build_source } = build_source;
    const { main_build } = android_ci_build_source;
    const { branch, build_id, target } = main_build;

    return {
      deviceId: name,
      branch,
      buildId: build_id,
      target,
    };
  }

  private hostToEnvList(host: Host): Environment[] {
    return host.groups.flatMap((group) => ({
      runtimeAlias: host.runtime,
      hostUrl: host.url,
      groupName: group.name,
      devices: group.cvds.map((cvd) => this.cvdToDevice(cvd)),
      status: EnvStatus.running,
    }));
  }

  private runtimeToEnvList(runtime: Runtime): Environment[] {
    return runtime.hosts.flatMap((host) => this.hostToEnvList(host));
  }

  private envsFromRuntimes$: Observable<EnvInitAction> = this.runtimeService
    .getRuntimes()
    .pipe(
      map((runtimes) =>
        runtimes.flatMap((runtime) => this.runtimeToEnvList(runtime))
      ),
      map((envs) => ({
        type: 'init',
        envs,
      }))
    );

  private isSame(env1: Environment, env2: Environment) {
    return env1.groupName === env2.groupName && env1.hostUrl === env2.hostUrl;
  }

  private environments: Observable<Environment[]> = merge(
    this.envsFromRuntimes$,
    this.envAction
  ).pipe(
    mergeScan((envs, action) => {
      if (action.type === 'init') {
        return of(action.envs);
      }

      if (action.type === 'delete') {
        return of(
          envs.map((env) => {
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
    }, [] as Environment[]),
    scan((oldEnvs, newEnvs) => {
      const oldStarting = oldEnvs.filter(
        (env) => env.status === EnvStatus.starting
      );

      const starting = oldStarting.filter(
        (oldEnv) => !newEnvs.find((newEnv) => this.isSame(oldEnv, newEnv))
      );

      const oldStopping = oldEnvs.filter(
        (env) => env.status === EnvStatus.stopping
      );

      const stoppingAndRunning = newEnvs.map((newEnv) => {
        if (oldStopping.find((oldEnv) => this.isSame(oldEnv, newEnv))) {
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
    private runtimeService: RuntimeService
  ) {}
}
