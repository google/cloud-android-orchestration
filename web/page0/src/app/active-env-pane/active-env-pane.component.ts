import {Component} from '@angular/core';
import {Store} from 'src/store/store';
import {EnvService} from '../env.service';
import {RefreshService} from '../refresh.service';
import {RuntimeViewStatus} from '../runtime-interface';

@Component({
  selector: 'app-active-env-pane',
  templateUrl: './active-env-pane.component.html',
  styleUrls: ['./active-env-pane.component.scss'],
})
export class ActiveEnvPaneComponent {
  envs$ = this.envService.getEnvs();
  status$ = this.store.select<RuntimeViewStatus>(
    store => store.runtimesLoadStatus
  );

  constructor(
    private envService: EnvService,
    private refreshService: RefreshService,
    private store: Store
  ) {}

  onClickRefresh() {
    this.refreshService.refresh();
  }

  showProgressBar(status: RuntimeViewStatus | null) {
    return (
      status === RuntimeViewStatus.refreshing ||
      status === RuntimeViewStatus.initializing
    );
  }
}
