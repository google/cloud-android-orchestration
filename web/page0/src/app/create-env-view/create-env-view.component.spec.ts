import {HttpTestingController} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {CreateEnvViewComponent} from './create-env-view.component';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {modules} from '../modules';
import {MatSelectHarness} from '@angular/material/select/testing';
import {Runtime, RuntimeStatus} from '../runtime-interface';
import {deriveApis, MockLocalStorage} from 'src/mock/apis';

describe('CreateEnvViewComponent', () => {
  function setUp() {
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

    const fixture = TestBed.createComponent(CreateEnvViewComponent);
    const component = fixture.componentInstance;
    const http = TestBed.inject(HttpTestingController);
    const loader = TestbedHarnessEnvironment.loader(fixture);

    runtimes.forEach(runtime => {
      const apis = deriveApis(runtime);
      for (const api of apis) {
        const {params, data, opts} = api;
        http.expectOne(params).flush(data, opts);
      }
    });

    return {component, loader, fixture, runtimes};
  }

  it('should create', () => {
    const {component} = setUp();
    expect(component).toBeTruthy();
  });

  it('localStorage set up', () => {
    const {runtimes} = setUp();
    const storedRuntimes = JSON.parse(window.localStorage.getItem('runtimes')!);
    expect(storedRuntimes.length).toBe(runtimes.length);
  });

  it('runtimes$ set up', done => {
    const {component} = setUp();

    component.runtimes$.subscribe(runtimes => {
      expect(runtimes.length).toBe(2);
      done();
    });
  });

  it('Open each selector without selecting any runtime', async () => {
    const {loader, runtimes} = setUp();

    const selectors = await loader.getAllHarnesses(MatSelectHarness);
    const [runtimeSelector, zoneSelector, hostSelector] = selectors;

    await runtimeSelector.open();
    const runtimeOptions = await runtimeSelector.getOptions();
    expect(runtimeOptions.length).toBe(runtimes.length + 1);

    for (let i = 0; i < runtimes.length; i++) {
      const optionText = await runtimeOptions[i].getText();
      expect(optionText).toBe(runtimes[i].alias);
    }

    await zoneSelector.open();
    const zoneOptions = await zoneSelector.getOptions();
    expect(zoneOptions.length).toBe(0);

    await hostSelector.open();
    const hostOptions = await hostSelector.getOptions();
    expect(hostOptions.length).toBe(1);
  });

  it('Press each selectors', async () => {
    const {loader, runtimes} = setUp();

    const selectors = await loader.getAllHarnesses(MatSelectHarness);
    const [runtimeSelector, zoneSelector, hostSelector] = selectors;

    await runtimeSelector.open();
    const runtimeOptions = await runtimeSelector.getOptions();

    for (let i = 0; i < runtimes.length; i++) {
      const runtime = runtimes[i];
      const runtimeOption = runtimeOptions[i];

      await runtimeOption.click();
      const zones = runtime.zones || [];

      await zoneSelector.open();
      const zoneOptions = await zoneSelector.getOptions();
      expect(zoneOptions.length).toBe(zones.length);

      for (let j = 0; j < zones.length; j++) {
        const zone = zones[j];
        const zoneOption = zoneOptions[j];

        await zoneOption.click();
        const hosts = runtime.hosts.filter(host => host.zone === zone);

        await hostSelector.open();

        const hostOptions = await hostSelector.getOptions();
        expect(hostOptions.length).toBe(hosts.length + 1);

        for (let k = 0; k < hosts.length; k++) {
          const host = hosts[k];
          const hostOption = hostOptions[k];

          const optionText = await hostOption.getText();
          expect(optionText).toBe(host.name);
        }
      }
    }
  });

  // TODO:

  it('Press "Register New" from Runtime selector', () => {});

  it('Press "Create New" from Runtime selector', () => {});

  it('Duplicate device #1', () => {});

  it('Add device #2 and delete device #2', () => {});

  it('Fill the form, add device #2, and press Create', () => {});

  it('Press Cancel', () => {});
});
