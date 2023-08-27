import {Component} from '@angular/core';
import {Store} from 'src/store/store';
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
  status$ = this.store.select<RuntimeViewStatus>(
    store => store.runtimesLoadStatus
  );

  constructor(
    private runtimeService: RuntimeService,
    private refreshService: RefreshService,
    private store: Store
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
