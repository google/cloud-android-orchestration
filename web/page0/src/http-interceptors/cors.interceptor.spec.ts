import {TestBed} from '@angular/core/testing';

import {CorsInterceptor} from './cors.interceptor';

describe('CorsInterceptor', () => {
  beforeEach(() =>
    TestBed.configureTestingModule({
      providers: [CorsInterceptor],
    })
  );

  it('should be created', () => {
    const interceptor: CorsInterceptor = TestBed.inject(CorsInterceptor);
    expect(interceptor).toBeTruthy();
  });
});
