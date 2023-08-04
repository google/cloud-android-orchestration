import { Component } from '@angular/core';
import { RuntimesStatus } from '../runtime-interface';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-list-runtime-view',
  templateUrl: './list-runtime-view.component.html',
  styleUrls: ['./list-runtime-view.component.scss'],
})
export class ListRuntimeViewComponent {
  runtimes$ = this.runtimeService.getRuntimes();
  status$ = this.runtimeService.getStatus();

  constructor(private runtimeService: RuntimeService) {}

  onClickRefresh() {
    this.runtimeService.refreshRuntimes();
  }

  showProgressBar(status: string | null) {
    return (
      status === RuntimesStatus.initializing ||
      status === RuntimesStatus.refreshing
    );
  }
}
