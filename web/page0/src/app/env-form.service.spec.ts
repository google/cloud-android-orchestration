import {HttpClientModule} from '@angular/common/http';
import {TestBed} from '@angular/core/testing';

import {EnvFormService} from './env-form.service';

describe('EnvFormService', () => {
  let service: EnvFormService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientModule],
    });
    service = TestBed.inject(EnvFormService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  // TODO:
  // getInitEnvForm
  // getEnvForm
  // runtimes, zones, hosts
  // getSelectedRuntime
  // getValue
  // clearForm
});
