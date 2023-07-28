import { Injectable } from '@angular/core';
import { Observable, of } from 'rxjs';
import { mockEnvList } from 'src/mock/envs';
import { Envrionment } from './env-interface';

@Injectable({
  providedIn: 'root'
})
export class EnvService {
  private environments: Observable<Envrionment[]> = of(mockEnvList)

  getEnvs() {
    return this.environments
  }

  constructor() { }
}
