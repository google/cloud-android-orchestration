import {HttpTestingController} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';

import {deriveApis, MockLocalStorage} from 'src/mock/apis';

import {EnvService} from './env.service';
import {modules} from './modules';
import {Runtime, RuntimeStatus} from './runtime-interface';

describe('EnvService', () => {
  function setUp() {
    const storedRuntimes: Runtime[] = [
      {
        alias: 'runtime1',
        type: 'cloud',
        url: 'http://runtime1.example.com/api',
        zones: ['zone1', 'zone2'],
        hosts: [
          {
            name: 'host1',
            zone: 'zone1',
            url: 'http://runtime1.example.com/api/v1/zones/zone1/hosts/host1',
            runtime: 'runtime1',
            groups: [
              {
                name: 'group1',
                cvds: [
                  {
                    name: 'cvd1',
                    build_source: {
                      android_ci_build_source: {
                        main_build: {
                          branch: 'example_branch',
                          build_id: 'example_build_id',
                          target: 'example_target',
                        },
                      },
                    },
                    status: 'valid',
                    displays: [],
                    group_name: 'group1',
                  },
                ],
              },
            ],
          },
        ],
        status: RuntimeStatus.valid,
      },
      {
        alias: 'runtime2',
        type: 'cloud',
        url: 'http://runtime2.example.com/api',
        zones: ['zone1', 'zone2'],
        hosts: [],
        status: RuntimeStatus.valid,
      },
    ];

    const mockLocalStorage = new MockLocalStorage({
      runtimes: storedRuntimes,
    });

    spyOn(window.localStorage, 'getItem').and.callFake(
      mockLocalStorage.getItem.bind(mockLocalStorage)
    );
    spyOn(window.localStorage, 'setItem').and.callFake(
      mockLocalStorage.setItem.bind(mockLocalStorage)
    );

    TestBed.configureTestingModule(modules);

    const service = TestBed.inject(EnvService);
    const http = TestBed.inject(HttpTestingController);

    return {service, storedRuntimes, http};
  }

  it('should be created', () => {
    const {service} = setUp();
    expect(service).toBeTruthy();
  });

  it('#getEnvs: get initial envs', done => {
    const {service, storedRuntimes, http} = setUp();
    const initEnvIdentifiers = new Set(
      storedRuntimes.flatMap(runtime => {
        if (!runtime.hosts) {
          return [];
        }

        return runtime.hosts.flatMap(host =>
          host.groups.map(group => `${runtime.alias}-${host.url}-${group.name}`)
        );
      })
    );

    service.getEnvs().subscribe(envs => {
      const envIdentifiers = new Set(
        envs.map(env => `${env.runtimeAlias}-${env.hostUrl}-${env.groupName}`)
      );

      expect(envIdentifiers.size).toBe(initEnvIdentifiers.size);
      expect([...envIdentifiers].every(id => initEnvIdentifiers.has(id))).toBe(
        true
      );

      done();
    });

    storedRuntimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        console.log(params.method, ' ', params.url);
        http.expectOne(params).flush(data, opts);
      }
    });
  });

  it('#createEnv: valid form', done => {
    const {service, storedRuntimes, http} = setUp();
    const exampleRuntime = storedRuntimes[0];
    const exampleHost = exampleRuntime.hosts[0];

    service
      .createEnv(exampleRuntime.alias, exampleHost.url, {
        groupName: 'new-group',
        devices: [
          {
            deviceId: 'new-cvd',
            branch: 'example_branch',
            target: 'example_target',
            buildId: 'example_build_id',
          },
        ],
      })
      .subscribe(res => {
        done();
      });

    http
      .expectOne({
        url: `${exampleHost.url}/cvds`,
        method: 'POST',
      })
      .flush({
        name: 'new-op',
        done: false,
      });
  });

  it('#createEnv: invalid form', done => {
    const {service, storedRuntimes, http} = setUp();
    const exampleRuntime = storedRuntimes[0];
    const exampleHost = exampleRuntime.hosts[0];

    service
      .createEnv(exampleRuntime.alias, exampleHost.url, {
        groupName: 'invalid-group',
        devices: [
          {
            deviceId: 'dup-cvd',
            branch: 'example_branch',
            target: 'example_target',
            buildId: 'example_build_id',
          },
          {
            deviceId: 'dup-cvd',
            branch: 'example_branch',
            target: 'example_target',
            buildId: 'example_build_id',
          },
        ],
      })
      .subscribe({
        next: () => done.fail('Should fail'),
        error: () => done(),
      });
  });
});
