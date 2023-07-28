import {Component, Injectable} from '@angular/core';
import {BUILD_VERSION} from '../version';

@Injectable()
@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
})
export class AppComponent {
  readonly version = BUILD_VERSION;
  constructor() {}
}
