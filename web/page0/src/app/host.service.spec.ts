import {HttpClientModule} from '@angular/common/http';
import {TestBed} from '@angular/core/testing';

import {HostService} from './host.service';

describe('HostService', () => {
  let service: HostService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientModule],
    });
    service = TestBed.inject(HostService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
