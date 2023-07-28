import {Component} from '@angular/core';
import {FormControl, Validators} from '@angular/forms';
import { Router } from '@angular/router';
import {BehaviorSubject, combineLatestWith, map, Observable, of} from 'rxjs';

@Component({
  selector: 'app-create-env-view',
  templateUrl: './create-env-view.component.html',
  styleUrls: ['./create-env-view.component.scss'],
})
export class CreateEnvViewComponent {
  constructor(private router: Router) {}

  groupIdFormControl = new FormControl('', [
    Validators.required,
    Validators.minLength(1),
  ]);

  start = of(1);
  actionSubject = new BehaviorSubject(0);
  action = this.actionSubject.asObservable();
  count = this.start.pipe(combineLatestWith(this.action)).pipe(
    map(([startingValue, action]) => {
      return startingValue + action;
    })
  );

  ngOnInit() {
    this.count.subscribe(count => console.log(count));
  }

  increment() {
    const currentValue = this.actionSubject.getValue();
    this.actionSubject.next(currentValue + 1);
  }

  decrement() {
    const currentValue = this.actionSubject.getValue();

    if (currentValue == 1) {
      return;
    }

    this.actionSubject.next(currentValue - 1);
  }

  getIndices(): Observable<Array<number>> {
    return this.count.pipe(
      map(count => {
        return [...Array(count).keys()];
      })
    );
  }

  onClickRegisterRuntime() {
    this.router.navigate(['/register-runtime'], {
      queryParams: {
        previousUrl: "create-env"
      }
    })
  }


  onClickCreateHost() {
    this.router.navigate(['/create-host'], {
      queryParams: {
        previousUrl: "create-env"
      }
    })
  }


  selectedRuntime = '';
}
