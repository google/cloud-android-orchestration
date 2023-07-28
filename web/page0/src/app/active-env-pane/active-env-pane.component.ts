import { Component } from '@angular/core';
import { EnvService } from '../env.service';

@Component({
  selector: 'app-active-env-pane',
  templateUrl: './active-env-pane.component.html',
  styleUrls: ['./active-env-pane.component.scss']
})
export class ActiveEnvPaneComponent {
  envs = this.envService.getEnvs()

  constructor(private envService: EnvService) {}
}
