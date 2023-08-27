import {Component} from '@angular/core';
import {envSelector} from 'src/store/selectors';
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
  envs$ = this.store.select(envSelector);
  status$ = this.store.select<RuntimeViewStatus>(
    store => store.runtimesLoadStatus
  );

  constructor(
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
