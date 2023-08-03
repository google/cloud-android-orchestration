import { Component } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { map, Observable, of } from 'rxjs';
import { RuntimeService } from '../runtime.service';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'app-register-runtime-view',
  templateUrl: './register-runtime-view.component.html',
  styleUrls: ['./register-runtime-view.component.scss'],
})
export class RegisterRuntimeViewComponent {
  constructor(
    private runtimeService: RuntimeService,
    private formBuilder: FormBuilder,
    private router: Router,
    private snackBar: MatSnackBar
  ) {}

  runtimes$ = this.runtimeService.getRuntimes();
  loading$: Observable<boolean> = of(true);

  runtimeForm = this.formBuilder.group({
    url: ['http://localhost:5000', Validators.required],
    alias: ['test', Validators.required],
  });

  onSubmit() {
    this.runtimeService
      .verifyRuntime(this.runtimeForm.value.url || '')
      .pipe(
        map((runtime) => {
          this.runtimeService.registerRuntime(runtime);
          this.router.navigate(['/list-runtime']);
        })
      )
      .subscribe({
        next: (data) => {
          this.snackBar.dismiss();
        },
        error: (error) => {
          this.snackBar.open(error.message);
        },
      });

    // TODO: show progress bar
  }
}
