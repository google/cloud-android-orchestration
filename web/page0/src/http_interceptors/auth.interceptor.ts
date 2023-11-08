import {Injectable} from '@angular/core';
import {
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpInterceptor,
  HttpErrorResponse,
} from '@angular/common/http';
import {Observable} from 'rxjs';
import {catchError} from 'rxjs/operators';
import {MatSnackBar} from '@angular/material/snack-bar';

const handleAuthError = (error: HttpErrorResponse, snackBar: MatSnackBar) => {
  snackBar.open(
    `Request failed: check your credentials (error message: ${error.message})`,
    'Dismiss'
  );
};

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  constructor(private snackBar: MatSnackBar) {}

  intercept(
    request: HttpRequest<unknown>,
    next: HttpHandler
  ): Observable<HttpEvent<unknown>> {
    return next.handle(request).pipe(
      catchError((error: HttpErrorResponse) => {
        // If credentials got expired, OPTIONS request fails before the actual request
        // Thus, error status becomes 0 for CORS failure
        if (error.status === 0) {
          handleAuthError(error, this.snackBar);
        }

        throw error;
      })
    );
  }
}
