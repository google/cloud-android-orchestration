import { TestBed } from '@angular/core/testing';

import { DeviceFormService } from './device-form.service';

describe('DeviceFormService', () => {
  let service: DeviceFormService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(DeviceFormService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
