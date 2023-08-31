import {HttpErrorResponse} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable, timer} from 'rxjs';
import {retry} from 'rxjs/operators';
import {ApiService} from './api.service';

@Injectable({
  providedIn: 'root',
})
export class OperationService {
  constructor(private apiService: ApiService) {}

  longPolling<T>(waitUrl: string): Observable<T> {
    const retryConfig = {
      count: 1000,
      delay: (err: HttpErrorResponse, retryCount: number) => {
        if (retryCount % 10 === 0) {
          console.warn(
            `Wait toward ${waitUrl} has failed for ${retryCount} times`
          );
        }

        if (err.status === 503) {
          return timer(0);
        }

        throw new Error(`Request ${err.url} failed to be done`);
      },
    };

    return this.apiService.wait<T>(waitUrl).pipe(retry(retryConfig));
  }
}
