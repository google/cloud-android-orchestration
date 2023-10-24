import {HttpErrorResponse} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {merge, Observable, timer} from 'rxjs';
import {map, mergeAll, retry, shareReplay, switchMap} from 'rxjs/operators';
import {ApiService} from './api.service';
import {Operation} from './interface/cloud-orchestrator.dto';
import {
  DoneResult,
  Result,
  ResultType,
  WaitStartedResult,
} from './interface/result-interface';

@Injectable({
  providedIn: 'root',
})
export class OperationService {
  constructor(private apiService: ApiService) {}

  private longPolling<T>(waitUrl: string): Observable<T> {
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

  wait<T>(
    request: Observable<Operation>,
    waitUrlSynthesizer: (op: Operation) => string
  ): Observable<Result<T>> {
    const requestReplay = request.pipe(shareReplay(1));

    const requestResult: Observable<WaitStartedResult> = requestReplay.pipe(
      map(operation => ({
        type: ResultType.waitStarted as ResultType.waitStarted,
        waitUrl: waitUrlSynthesizer(operation),
      }))
    );

    const waitResult: Observable<DoneResult<T>> = requestReplay.pipe(
      switchMap(operation => {
        const waitUrl = waitUrlSynthesizer(operation);
        return this.longPolling<T>(waitUrl).pipe(
          map(data => ({
            type: ResultType.done as ResultType.done,
            data,
            waitUrl,
          }))
        );
      })
    );

    return merge([requestResult, waitResult]).pipe(mergeAll());
  }
}
