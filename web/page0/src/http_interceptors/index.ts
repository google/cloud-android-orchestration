import {HTTP_INTERCEPTORS} from '@angular/common/http';
import {AuthInterceptor} from './auth.interceptor';
import {CorsInterceptor} from './cors.interceptor';

export const httpInterceptorProviders = [
  {provide: HTTP_INTERCEPTORS, useClass: CorsInterceptor, multi: true},
  {provide: HTTP_INTERCEPTORS, useClass: AuthInterceptor, multi: true},
];
