import {
  afterRenderEffect,
  Component,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  input,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { RouterLink } from '@angular/router';

import { RedocHandle, RedocService } from '../../../../services/redoc.service';
import { ThemeService } from '../../../../services/theme.service';
import { ApiInfo, specPath } from '../../../../utils/apis';

type Status = 'loading' | 'ready' | 'failed';

@Component({
  imports: [RouterLink],
  selector: 'app-redoc-view',
  styleUrl: './redoc-view.scss',
  template: `
    @switch (status()) {
      @case ('loading') {
        <p class="status" role="status">Loading the {{ api().name }} reference…</p>
      }
      @case ('failed') {
        <p class="status" role="alert">
          The interactive reference could not be loaded.
          <a routerLink="." [queryParams]="{ view: 'source' }">Read the specification source</a>
          instead.
        </p>
      }
    }
    <div #host class="redoc" [hidden]="status() === 'failed'"></div>
  `,
})
export class RedocView {
  private readonly redoc = inject(RedocService);
  private readonly theme = inject(ThemeService);
  private readonly host = viewChild.required<ElementRef<HTMLElement>>('host');

  readonly api = input.required<ApiInfo>();
  /** Distance from the top of the viewport to the bottom of whatever is pinned above Redoc. */
  readonly scrollOffset = input<() => number>(() => 0);

  protected readonly status = signal<Status>('loading');
  private run = 0;
  private rendered?: RedocHandle;

  constructor() {
    // Redoc has its own light and dark themes: follow the site's choice. Runs before the first
    // render below, so Redoc starts in the right mode.
    effect(() => this.redoc.setColorMode(this.theme.scheme()));
    afterRenderEffect({
      write: () => {
        const api = this.api();
        const host = this.host().nativeElement;
        untracked(() => this.render(api, host));
      },
    });
    inject(DestroyRef).onDestroy(() => {
      this.run++;
      this.clear();
    });
  }

  private async render(api: ApiInfo, host: HTMLElement): Promise<void> {
    // Ignore a slow render that finishes after the user moved on to another API.
    const run = ++this.run;
    this.status.set('loading');
    this.clear();
    // A fresh element per render: React must not reuse a container it has unmounted from.
    const element = host.ownerDocument.createElement('div');
    host.replaceChildren(element);
    try {
      const rendered = await this.redoc.render(specPath(api), element, {
        scrollYOffset: this.scrollOffset(),
        hideDownloadButtons: true,
        disableTelemetry: true,
      });
      if (run !== this.run) {
        rendered.destroy();
        return;
      }
      this.rendered = rendered;
      this.status.set('ready');
    } catch (error) {
      console.error(error);
      if (run === this.run) this.status.set('failed');
    }
  }

  private clear(): void {
    this.rendered?.destroy();
    this.rendered = undefined;
  }
}
