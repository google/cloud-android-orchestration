import {HttpErrorResponse} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable, of, throwError} from 'rxjs';
import {retry} from 'rxjs/operators';
import {ApiService} from './api.service';

@Injectable({
  providedIn: 'root',
})
export class OperationService {
  constructor(private apiService: ApiService) {}

  longPolling<T>(waitUrl: string): Observable<T> {
    return this.apiService.wait<T>(waitUrl).pipe(
      retry({
        delay: (err: HttpErrorResponse, retryCount) => {
          if (retryCount % 10 === 0) {
            console.warn(
              `Wait toward ${waitUrl} has failed for ${retryCount} times`
            );
          }

          if (err.status === 503) {
            return of(1);
          }

          return throwError(
            () => new Error(`Request ${err.url} failed to be done`)
          );
        },
      })
    );
  }
}
