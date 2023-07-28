import { HttpClient } from '@angular/common/http';
import { Component } from '@angular/core';
import { Subject } from 'rxjs';

@Component({
  selector: 'app-register-runtime-view',
  templateUrl: './register-runtime-view.component.html',
  styleUrls: ['./register-runtime-view.component.scss']
})
export class RegisterRuntimeViewComponent {
  url: string = ""

  constructor(private httpClient: HttpClient) {}

  onChangeURL(event: Event) {
    this.url = (event?.target as HTMLInputElement)?.value
    console.log("URL ", this.url)
  }

  checkValidity(url: string) {
    console.log(`checking validity of ${url}`)
    return this.httpClient.get(`${url}/status`)
  }

  registerSubject: Subject<void> = new Subject()


  onClickRegister() {
    // TODO: get value from URL
    // TODO: send request to the {URL}/status
    this.registerSubject.next()
    this.checkValidity(this.url).pipe()
  }

}
