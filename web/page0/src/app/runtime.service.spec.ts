import { TestBed } from '@angular/core/testing';

import { RuntimeService } from './runtime.service';

describe('RuntimeService', () => {
  let service: RuntimeService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(RuntimeService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
