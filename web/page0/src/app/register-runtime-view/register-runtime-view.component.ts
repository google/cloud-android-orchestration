import { Component } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { RuntimeService } from '../runtime.service';
import { MatSnackBar } from '@angular/material/snack-bar';
import { RuntimesStatus } from '../runtime-interface';

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
  ) {
    this.status$.subscribe((status) => console.log(`status: ${status}`));
  }

  runtimes$ = this.runtimeService.getRuntimes();
  status$ = this.runtimeService.getStatus();

  runtimeForm = this.formBuilder.group({
    url: ['http://localhost:3000', Validators.required],
    alias: ['test', Validators.required],
  });

  showProgressBar(status: string | null) {
    return status === RuntimesStatus.registering;
  }

  onSubmit() {
    const url = this.runtimeForm.value.url;
    const alias = this.runtimeForm.value.alias;

    if (!url || !alias) {
      return;
    }

    this.runtimeService.registerRuntime(alias, url).subscribe({
      next: () => {
        this.router.navigate(['/list-runtime']);
        this.snackBar.dismiss();
      },
      error: (error) => {
        this.snackBar.open(error.message);
      },
    });
  }
}
