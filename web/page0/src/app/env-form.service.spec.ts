import { TestBed } from '@angular/core/testing';

import { EnvFormService } from './env-form.service';

describe('EnvFormService', () => {
  let service: EnvFormService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(EnvFormService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
