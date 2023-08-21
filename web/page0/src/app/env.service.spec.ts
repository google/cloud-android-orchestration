import {HttpClientModule} from '@angular/common/http';
import {TestBed} from '@angular/core/testing';

import {EnvService} from './env.service';

describe('EnvService', () => {
  let service: EnvService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [],
      imports: [HttpClientModule],
    });
    service = TestBed.inject(EnvService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
