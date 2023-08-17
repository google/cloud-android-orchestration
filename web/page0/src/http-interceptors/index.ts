import {HTTP_INTERCEPTORS} from '@angular/common/http';
import {CorsInterceptor} from './cors.interceptor';

export const httpInterceptorProviders = [
  {provide: HTTP_INTERCEPTORS, useClass: CorsInterceptor, multi: true},
];
