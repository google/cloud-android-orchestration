import {Component} from '@angular/core';
import {
  envCardListSelector,
  runtimesLoadStatusSelector,
} from 'src/app/store/selectors';
import {Store} from 'src/app/store/store';
import {RefreshService} from '../refresh.service';
import {RuntimeViewStatus} from 'src/app/interface/runtime-interface';

@Component({
  standalone: false,
  selector: 'app-active-env-pane',
  templateUrl: './active-env-pane.component.html',
  styleUrls: ['./active-env-pane.component.scss'],
})
export class ActiveEnvPaneComponent {
  envs$;
  status$;

  constructor(
    private refreshService: RefreshService,
    private store: Store
  ) {
    this.envs$ = this.store.select(envCardListSelector);
    this.status$ = this.store.select(runtimesLoadStatusSelector);
  }

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
