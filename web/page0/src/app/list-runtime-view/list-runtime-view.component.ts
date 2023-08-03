import { Component } from '@angular/core';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-list-runtime-view',
  templateUrl: './list-runtime-view.component.html',
  styleUrls: ['./list-runtime-view.component.scss'],
})
export class ListRuntimeViewComponent {
  runtimes$ = this.runtimeService.getRuntimes();

  constructor(private runtimeService: RuntimeService) {}

  onClickRefresh() {
    this.runtimeService.refreshRuntimes();
  }
}
