import {HttpTestingController} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {lastValueFrom, take} from 'rxjs';
import {deriveApis, MockLocalStorage} from 'src/mock/apis';
import {modules} from './modules';
import {Runtime, RuntimeStatus} from './runtime-interface';
import {RuntimeService} from './runtime.service';

describe('RuntimeService', () => {
  async function setUp() {
    const runtimes: Runtime[] = [
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
      runtimes,
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

    return {service, runtimes, http};
  }

  fit('should be created', async () => {
    const {service} = await setUp();
    expect(service).toBeTruthy();
  });

  fit('default runtimes should be load', async () => {
    const {service, runtimes, http} = await setUp();

    service.getRuntimes().subscribe(runtimes => {
      console.log(runtimes);
    });

    runtimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }
    });

    // const initialRuntimes = await lastValueFrom(
    //   service.getRuntimes().pipe(take(1))
    // );
    // expect(initialRuntimes.length).toBe(runtimes.length);
  });

  it('#getRuntimeInfo', async done => {
    const {service} = await setUp();

    service
      .getRuntimeInfo('http://localhost:8071/api', 'default')
      .subscribe(runtime => {
        expect(runtime.zones?.length).toBe(2);
        done();
      });
  });

  it('#register', async done => {
    const {service} = await setUp();

    const NEW_RUNTIME_ALIAS = 'new_runtime';
    const NEW_RUNTIME_URL = 'http://localhost:8071/api';

    service
      .registerRuntime(NEW_RUNTIME_ALIAS, NEW_RUNTIME_URL)
      .subscribe(runtime => {
        expect(runtime.alias).toBe(NEW_RUNTIME_ALIAS);
        expect(runtime.url).toBe(NEW_RUNTIME_URL);
        done();
      });
  });

  it('#register duplicate', async done => {
    const {service} = await setUp();

    const DUPLICATE_RUNTIME_ALIAS = 'runtime1';
    const DUPLICATE_RUNTIME_URL = 'http://localhost:8071/api';

    service
      .registerRuntime(DUPLICATE_RUNTIME_ALIAS, DUPLICATE_RUNTIME_URL)
      .subscribe({
        next: runtime => done.fail('Should fail with duplicate runtime error'),
        error: () => {
          done();
        },
      });
  });

  it('#unregister', async () => {
    const {service} = await setUp();
    service.unregisterRuntime('runtime1');

    const runtimes = await lastValueFrom(service.getRuntimes().pipe(take(2)));

    expect(runtimes.length).toBe(1);
  });

  // it('#unregister', fakeAsync(async () => {
  //   const { service } = setUp();
  //   service.unregisterRuntime('runtime1');
  //   service.getRuntimes().subscribe((runtimes) => {
  //     expect(runtimes.length).toBe(1);
  //     expect(runtimes[0].alias === 'runtime2');
  //   });
  // }));
});
