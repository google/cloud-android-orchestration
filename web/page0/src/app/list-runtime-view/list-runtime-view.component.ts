import { Component } from '@angular/core';
import { BehaviorSubject } from 'rxjs';
import { RuntimeService } from '../runtime.service';

@Component({
  selector: 'app-list-runtime-view',
  templateUrl: './list-runtime-view.component.html',
  styleUrls: ['./list-runtime-view.component.scss'],
})
export class ListRuntimeViewComponent {
  runtimes$ = this.runtimeService.getRuntimes();
  loading$ = new BehaviorSubject<boolean>(false);

  constructor(private runtimeService: RuntimeService) {}

  ngOnInit() {
    this.runtimeService.initRuntimes(this.loading$);
  }

  onClickRefresh() {
    this.runtimeService.refreshRuntimes(this.loading$);
  }
}
