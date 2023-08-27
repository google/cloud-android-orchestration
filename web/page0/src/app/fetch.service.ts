import {Injectable} from '@angular/core';
import {forkJoin, Observable, of} from 'rxjs';
import {
  map,
  catchError,
  mergeMap,
  defaultIfEmpty,
  switchMap,
} from 'rxjs/operators';
import {ApiService} from './api.service';
import {Host} from './host-interface';
import {Runtime, RuntimeStatus} from './runtime-interface';

@Injectable({
  providedIn: 'root',
})
export class FetchService {
  private fetchGroups(hostUrl: string) {
    return this.apiService.listGroups(hostUrl).pipe(map(({groups}) => groups));
  }

  private fetchHosts(
    runtimeUrl: string,
    zone: string,
    runtimeAlias: string
  ): Observable<Host[]> {
    return this.apiService.listHosts(runtimeUrl, zone).pipe(
      map(({items: hosts}) => hosts || []),
      mergeMap(hosts => {
        return forkJoin(
          hosts.map(host => {
            const hostUrl = `${runtimeUrl}/v1/zones/${zone}/hosts/${host.name}`;
            return this.fetchGroups(hostUrl).pipe(
              map(groups => ({
                name: host.name!,
                zone: zone,
                url: hostUrl,
                runtime: runtimeAlias,
                groups,
              }))
            );
          })
        ).pipe(defaultIfEmpty([]));
      })
    );
  }

  fetchRuntimeInfo(url: string, alias: string): Observable<Runtime> {
    return this.apiService.getRuntimeInfo(url).pipe(
      switchMap(info => {
        // TODO: handle local workstation depending on type
        return this.apiService.listZones(url).pipe(
          map(({items: zones}) => zones || []),
          map(zones => ({
            type: info.type,
            zones: zones.map(zone => zone.name),
          }))
        );
      }),
      switchMap(({type, zones}) => {
        return forkJoin(
          zones.map(zone => this.fetchHosts(url, zone, alias))
        ).pipe(
          defaultIfEmpty([]),
          map(hostLists => hostLists.flat()),
          map((hosts: Host[]) => ({
            alias,
            type,
            url,
            zones,
            hosts,
            status: RuntimeStatus.valid,
          }))
        );
      }),

      catchError(error => {
        console.error(`Error from fetchRuntimeInfo of: ${alias} (${url})`);
        console.error(error);
        return of({
          alias,
          url,
          hosts: [],
          status: RuntimeStatus.error,
        });
      })
    );
  }

  constructor(private apiService: ApiService) {}
}