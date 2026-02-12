import {Component, ChangeDetectionStrategy} from '@angular/core';

@Component({
  changeDetection: ChangeDetectionStrategy.Default,
  standalone: false,
  selector: 'app-env-list-view',
  templateUrl: './env-list-view.component.html',
  styleUrls: ['./env-list-view.component.scss'],
})
export class EnvListViewComponent {
  constructor() {}

  ngOnInit() {}
}
