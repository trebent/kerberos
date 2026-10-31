import { Component, DestroyRef, inject, input, signal } from '@angular/core';
import { Clipboard } from '@angular/cdk/clipboard';
import { MatButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';

type CopyState = 'idle' | 'copied' | 'failed';

const RESET_AFTER_MS = 1500;

/** A small "Copy" button for the dark code panels. Announces the result to screen readers. */
@Component({
  imports: [MatButton, MatIcon],
  selector: 'app-copy-button',
  styleUrl: './copy-button.scss',
  template: `
    <button matButton type="button" (click)="copy()">
      <mat-icon aria-hidden="true">{{ icons[state()] }}</mat-icon>
      Copy<span class="visually-hidden"> {{ subject() }}</span>
    </button>
    <span class="visually-hidden" role="status">
      @switch (state()) {
        @case ('copied') {
          Copied to the clipboard
        }
        @case ('failed') {
          Copying failed
        }
      }
    </span>
  `,
})
export class CopyButton {
  private readonly clipboard = inject(Clipboard);
  private timer?: ReturnType<typeof setTimeout>;

  readonly text = input.required<string>();
  /** What is copied, read after "Copy" by screen readers, e.g. "the commands". */
  readonly subject = input('');

  protected readonly state = signal<CopyState>('idle');
  protected readonly icons: Record<CopyState, string> = {
    idle: 'content_copy',
    copied: 'check',
    failed: 'error',
  };

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
  }

  protected copy(): void {
    this.state.set(this.clipboard.copy(this.text()) ? 'copied' : 'failed');
    clearTimeout(this.timer);
    this.timer = setTimeout(() => this.state.set('idle'), RESET_AFTER_MS);
  }
}
