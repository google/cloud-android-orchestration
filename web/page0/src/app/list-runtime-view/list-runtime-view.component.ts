import {Component} from '@angular/core';
import {RefreshService} from '../refresh.service';
import {RuntimeViewStatus} from '../runtime-interface';
import {RuntimeService} from '../runtime.service';

@Component({
  selector: 'app-list-runtime-view',
  templateUrl: './list-runtime-view.component.html',
  styleUrls: ['./list-runtime-view.component.scss'],
})
export class ListRuntimeViewComponent {
  runtimes$ = this.runtimeService.getRuntimes();
  status$ = this.runtimeService.getStatus();

  constructor(
    private runtimeService: RuntimeService,
    private refreshService: RefreshService
  ) {}

  onClickRefresh() {
    this.refreshService.refresh();
  }

  showProgressBar(status: string | null) {
    return (
      status === RuntimeViewStatus.initializing ||
      status === RuntimeViewStatus.refreshing
    );
  }
}
