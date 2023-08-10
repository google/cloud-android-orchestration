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
import { Environment, EnvStatus } from './env-interface';
import { Host } from './host-interface';
import { Runtime } from './runtime-interface';
import { RuntimeService } from './runtime.service';
import { hasDuplicate } from './utils';

interface EnvCreateAction {
  type: 'create';
  env: Environment;
}

interface EnvDeleteAction {
  type: 'delete';
  groupName: string;
  hostUrl: string;
}

type EnvAction = EnvCreateAction | EnvDeleteAction;

@Injectable({
  providedIn: 'root',
})
export class EnvService {
  private envAction = new Subject<EnvAction>();

  createEnv(
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

    return this.apiService.createGroup(hostUrl, {
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
    });
  }

  deleteEnv(hostUrl: string, groupName: string) {
    // TODO: long polling
    return this.apiService.deleteGroup(hostUrl, groupName);
  }

  private hostToEnvList(host: Host): Environment[] {
    return host.groups.flatMap((group) => ({
      runtimeAlias: host.runtime,
      hostUrl: host.url,
      groupName: group.name,
      devices: group.cvds.map((cvd) => cvd.name),
      status: EnvStatus.running,
    }));
  }

  private runtimeToEnvList(runtime: Runtime): Environment[] {
    return runtime.hosts.flatMap((host) => this.hostToEnvList(host));
  }

  private envsFromRuntimes$: Observable<Environment[]> = this.runtimeService
    .getRuntimes()
    .pipe(
      map((runtimes) =>
        runtimes.flatMap((runtime) => this.runtimeToEnvList(runtime))
      )
    );

  private envsFromActions$: Observable<Environment[]> = this.envAction.pipe(
    tap((action) => console.log('env: ', action)),
    mergeScan((envs, action) => {
      return of(envs);
    }, [] as Environment[])
  );

  private isSame(env1: Environment, env2: Environment) {
    return (
      env1.runtimeAlias === env2.runtimeAlias &&
      env1.groupName === env2.groupName &&
      env1.hostUrl === env2.hostUrl
    );
  }

  private environments: Observable<Environment[]> = merge(
    this.envsFromRuntimes$,
    this.envsFromActions$
  ).pipe(
    scan((oldEnvs, newEnvs) => {
      const oldStarting = oldEnvs.filter(
        (env) => env.status === EnvStatus.starting
      );

      const starting = oldStarting.filter((oldEnv) =>
        newEnvs.find((newEnv) => this.isSame(oldEnv, newEnv))
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
