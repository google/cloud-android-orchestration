import { Injectable } from '@angular/core';
import {
  forkJoin,
  map,
  mergeScan,
  Observable,
  of,
  shareReplay,
  startWith,
  Subject,
  switchMap,
  tap,
} from 'rxjs';
import { ApiService } from './api.service';
import { Envrionment, EnvStatus } from './env-interface';
import { Host } from './host-interface';
import { HostService } from './host.service';
import { hasDuplicate } from './utils';

interface EnvCreateAction {
  type: 'create';
  env: Envrionment;
}

interface EnvDeleteAction {
  type: 'delete';
  groupName: string;
  hostUrl: string;
}

interface EnvInitAction {
  type: 'init';
}

interface EnvRefreshAction {
  type: 'refresh';
}

type EnvAction =
  | EnvCreateAction
  | EnvDeleteAction
  | EnvInitAction
  | EnvRefreshAction;

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

  private listEnvs(host: Host): Observable<Envrionment[]> {
    return this.apiService.listGroups(host.url).pipe(
      map(({ groups }) =>
        groups.flatMap((group) => {
          return {
            runtimeAlias: host.runtime,
            hostUrl: host.url,
            groupName: group.name,
            devices: group.cvds.map((cvd) => cvd.name),
            status: EnvStatus.running,
          };
        })
      )
    );
  }

  // get env from runtime
  private environments: Observable<Envrionment[]> = this.envAction.pipe(
    startWith({ type: 'init' } as EnvInitAction),
    tap((action) => console.log('env: ', action)),
    mergeScan((acc, action) => {
      if (action.type === 'init' || action.type === 'refresh') {
        return this.hostService.getAllHosts().pipe(
          switchMap((hosts) =>
            forkJoin(hosts.map((host) => this.listEnvs(host)))
          ),
          map((envLists) => envLists.flat())
        );
      }

      return of(acc);
    }, [] as Envrionment[]),
    shareReplay(1)
  );

  getEnvs() {
    return this.environments;
  }

  constructor(
    private apiService: ApiService,
    private hostService: HostService
  ) {}
}
