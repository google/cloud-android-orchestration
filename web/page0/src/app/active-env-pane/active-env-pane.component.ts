import {Component} from '@angular/core';
import {EnvService} from '../env.service';
import {RefreshService} from '../refresh.service';
import {RuntimeViewStatus} from '../runtime-interface';
import {RuntimeService} from '../runtime.service';

@Component({
  selector: 'app-active-env-pane',
  templateUrl: './active-env-pane.component.html',
  styleUrls: ['./active-env-pane.component.scss'],
})
export class ActiveEnvPaneComponent {
  envs$ = this.envService.getEnvs();
  status$ = this.runtimeService.getStatus();

  constructor(
    private envService: EnvService,
    private runtimeService: RuntimeService,
    private refreshService: RefreshService
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
