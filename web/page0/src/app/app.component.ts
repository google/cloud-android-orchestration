import {Component, Injectable, ChangeDetectionStrategy} from '@angular/core';
import {BUILD_VERSION} from '../version';
import {RefreshService} from './refresh.service';

@Injectable()
@Component({
  changeDetection: ChangeDetectionStrategy.Default,
  standalone: false,
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
})
export class AppComponent {
  readonly version = BUILD_VERSION;
  constructor(private refreshService: RefreshService) {
    this.refreshService.refresh();
  }
}
