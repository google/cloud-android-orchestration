import {Component, Input} from '@angular/core';
import {Envrionment, LocalEnvironment, RemoteEnvironment} from '../env-interface';

@Component({
  selector: 'app-env-card',
  templateUrl: './env-card.component.html',
  styleUrls: ['./env-card.component.scss'],
})
export class EnvCardComponent {
  @Input() env: Envrionment | null = null;
  remoteEnv: RemoteEnvironment | null = null;
  localEnv: LocalEnvironment | null = null;

  constructor() {}

  ngOnInit() {
    if (this.env?.env_type === "local") {
      this.localEnv = this.env as LocalEnvironment
    }

    if (this.env?.env_type === "remote") {
      this.remoteEnv = this.env as RemoteEnvironment
    }

    status = this.env?.status || "starting"

    if (status === "running") {
      this.tooltip = "Running"
      this.icon = "check_circle"
      return
    }

    if (status === "stopping") {
      this.tooltip = "Stopping"
      this.icon = "stop_circle"
      return
    }

    if (status === "error") {
      this.tooltip = "error"
      this.icon = "error"
      return
    }
 
    this.tooltip = "Starting"
    this.icon = "pending"
  }

  tooltip = ""
  icon = ""


  getColor() {
    if (this.env?.runtime === 'GCE') {
      return 'aliceblue';
    }

    if (this.env?.runtime === 'ARM') {
      return 'green';
    }

    return '#cfcfcf';
  }

  toggleFavorite() {
    
  }

  goToPerGroupUI() {
    window.open("https://google.com", "_blank")
  }

  /*
    TODOs

  */
}
