import {HttpTestingController} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {shareReplay, take} from 'rxjs/operators';
import {TestScheduler} from 'rxjs/testing';
import {deriveApis, invalidRuntime, MockLocalStorage} from 'src/mock/apis';
import {modules} from './modules';
import {Runtime, RuntimeStatus, RuntimeViewStatus} from './runtime-interface';

import {RuntimeService} from './runtime.service';

const testScheduler = new TestScheduler((actual, expected) => {
  expect(actual).toEqual(expected);
});

describe('RuntimeService', () => {
  function setUp() {
    const storedRuntimes: Runtime[] = [
      {
        alias: 'runtime1',
        type: 'cloud',
        url: 'http://runtime1.example.com/api',
        zones: ['zone1', 'zone2'],
        hosts: [],
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

    const service = TestBed.inject(RuntimeService);
    const http = TestBed.inject(HttpTestingController);

    return {service, storedRuntimes, http};
  }

  it('should be created', () => {
    const {service} = setUp();
    expect(service).toBeTruthy();
  });

  it('#getRuntimes: initial load', done => {
    const {service, storedRuntimes, http} = setUp();

    service.getRuntimes().subscribe(runtimes => {
      console.log(runtimes);
      expect(runtimes.length).toBe(storedRuntimes.length);
      done();
    });

    storedRuntimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }
    });
  });

  it('#getRuntimeInfo: valid runtime', done => {
    const {service, storedRuntimes, http} = setUp();
    const validRuntime = storedRuntimes[0];
    const {alias, url} = validRuntime;

    service.getRuntimeInfo(url, alias).subscribe(runtime => {
      expect(runtime.status).toBe(RuntimeStatus.valid);
      expect(runtime.zones?.length).toBe(validRuntime.zones?.length);
      expect(runtime.hosts?.length).toBe(validRuntime.hosts?.length);
      done();
    });

    const apis = deriveApis(validRuntime);
    for (const api of apis) {
      const {params, data, opts} = api;
      http.expectOne(params).flush(data, opts);
    }
  });

  it('#getRuntimeInfo: invalid runtime', done => {
    const {service, storedRuntimes, http} = setUp();
    const {alias, url} = invalidRuntime;
    expect(!storedRuntimes.find(r => r.alias === alias)).toBe(true);

    service.getRuntimeInfo(url, alias).subscribe(runtime => {
      expect(runtime.status).toBe(RuntimeStatus.error);
      done();
    });

    const apis = deriveApis(invalidRuntime);
    for (const api of apis) {
      const {params, data, opts} = api;
      http.expectOne(params).flush(data, opts);
    }
  });

  it('#register: valid runtime', done => {
    const {service, storedRuntimes, http} = setUp();

    testScheduler.run(({expectObservable}) => {
      const status$ = service.getStatus().pipe(shareReplay(5));
      status$.subscribe();

      const newRuntime = {
        alias: 'NEW_RUNTIME',
        type: 'cloud',
        url: 'http://new-runtime.example.com/api',
        zones: ['zone1'],
        hosts: [],
        status: RuntimeStatus.valid,
      } as Runtime;

      const {alias, url} = newRuntime;

      service
        .getRuntimes()
        .pipe(take(1))
        .subscribe({
          error: err => done.fail(err),
          next: runtimes => {
            const hasDuplicate = !!runtimes.find(
              runtime => runtime.alias === alias
            );

            expect(hasDuplicate).toBe(false);
          },
        });

      storedRuntimes.forEach(runtime => {
        const apis = deriveApis(runtime);
        for (const api of apis) {
          const {params, data, opts} = api;
          http.expectOne(params).flush(data, opts);
        }
      });

      service.registerRuntime(alias, url).subscribe(runtime => {
        expect(runtime.alias).toBe(alias);
        expect(runtime.url).toBe(url);
        done();
      });

      const apis = deriveApis(newRuntime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }

      expectObservable(status$).toBe('(iidrd)', {
        i: RuntimeViewStatus.initializing,
        d: RuntimeViewStatus.done,
        r: RuntimeViewStatus.registering,
      });
    });
  });

  it('#register: duplicated runtime', done => {
    const {service, storedRuntimes, http} = setUp();
    const dupRuntime = storedRuntimes[0];
    const {alias, url} = dupRuntime;

    testScheduler.run(({expectObservable}) => {
      const status$ = service.getStatus().pipe(shareReplay(5));
      status$.subscribe();

      service
        .getRuntimes()
        .pipe(take(1))
        .subscribe({
          error: err => done.fail(err),
          next: runtimes => {
            const hasDuplicate = !!runtimes.find(
              runtime => runtime.alias === alias
            );

            expect(hasDuplicate).toBe(true);
          },
        });

      storedRuntimes.forEach(runtime => {
        const apis = deriveApis(runtime);
        for (const api of apis) {
          const {params, data, opts} = api;
          http.expectOne(params).flush(data, opts);
        }
      });

      service.registerRuntime(alias, url).subscribe({
        error: () => done(),
        next: () => done.fail('Should emit error'),
      });

      const apis = deriveApis(dupRuntime);
      for (const api of apis) {
        const {params} = api;
        http.expectNone(params);
      }

      expectObservable(status$).toBe('(iidre)', {
        i: RuntimeViewStatus.initializing,
        d: RuntimeViewStatus.done,
        r: RuntimeViewStatus.registering,
        e: RuntimeViewStatus.register_error,
      });
    });
  });

  it('#unregister', done => {
    const {service, storedRuntimes, http} = setUp();
    const toBeRemovedRuntime = storedRuntimes[0];
    const {alias} = toBeRemovedRuntime;

    service
      .getRuntimes()
      .pipe(take(1))
      .subscribe({
        error: err => done.fail(err),
        next: runtimes => {
          const hasRuntime = !!runtimes.find(
            runtime => runtime.alias === alias
          );
          expect(hasRuntime).toBe(true);
        },
      });

    storedRuntimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }
    });

    service.unregisterRuntime(alias);

    service
      .getRuntimes()
      .pipe(take(1))
      .subscribe({
        next: runtimes => {
          const hasRuntime = !!runtimes.find(
            runtime => runtime.alias === alias
          );
          expect(hasRuntime).toBe(false);
          done();
        },
        error: err => done.fail(err),
      });
  });

  it('#refreshRuntimes', done => {
    const {service, storedRuntimes, http} = setUp();
    testScheduler.run(({expectObservable}) => {
      const status$ = service.getStatus().pipe(shareReplay(5));
      status$.subscribe();

      service
        .getRuntimes()
        .pipe(take(1))
        .subscribe({
          error: err => done.fail(err),
        });

      storedRuntimes.forEach(runtime => {
        const apis = deriveApis(runtime);
        for (const api of apis) {
          const {params, data, opts} = api;
          http.expectOne(params).flush(data, opts);
        }
      });

      service.refreshRuntimes();

      service
        .getRuntimes()
        .pipe(take(1))
        .subscribe({
          error: err => done.fail(err),
          next: () => done(),
        });

      storedRuntimes.forEach(runtime => {
        const apis = deriveApis(runtime);
        for (const api of apis) {
          const {params, data, opts} = api;
          http.expectOne(params).flush(data, opts);
        }
      });

      expectObservable(status$).toBe('(iidfd)', {
        i: RuntimeViewStatus.initializing,
        d: RuntimeViewStatus.done,
        f: RuntimeViewStatus.refreshing,
      });
    });
  });

  it('#getRuntimeByAlias: exists', done => {
    const {service, storedRuntimes, http} = setUp();
    const targetRuntime = storedRuntimes[0];
    const {alias, url} = targetRuntime;

    service.getRuntimeByAlias(alias).subscribe({
      next: runtime => {
        expect(runtime.url).toBe(url);
        done();
      },
      error: err => done.fail(err),
    });

    storedRuntimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }
    });
  });

  it('#getRuntimeByAlias: does not exist', done => {
    const {service, storedRuntimes, http} = setUp();
    const {alias} = invalidRuntime;
    expect(!storedRuntimes.find(r => r.alias === alias)).toBe(true);

    service.getRuntimeByAlias(alias).subscribe({
      error: () => done(),
      next: () => done.fail('Should emit error'),
    });

    storedRuntimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }
    });
  });
});
