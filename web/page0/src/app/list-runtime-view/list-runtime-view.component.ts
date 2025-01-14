import {Component} from '@angular/core';
import {Store} from 'src/app/store/store';
import {RefreshService} from '../refresh.service';
import {RuntimeViewStatus} from 'src/app/interface/runtime-interface';
import {RuntimeService} from '../runtime.service';
import {runtimesLoadStatusSelector} from '../store/selectors';

@Component({
  standalone: false,
  selector: 'app-list-runtime-view',
  templateUrl: './list-runtime-view.component.html',
  styleUrls: ['./list-runtime-view.component.scss'],
})
export class ListRuntimeViewComponent {
  runtimes$;
  status$;

  constructor(
    private runtimeService: RuntimeService,
    private refreshService: RefreshService,
    private store: Store
  ) {
    this.runtimes$ = this.runtimeService.getRuntimes();
    this.status$ = this.store.select(runtimesLoadStatusSelector);
  }

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
