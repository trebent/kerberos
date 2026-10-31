import {
  afterRenderEffect,
  Component,
  ElementRef,
  inject,
  input,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { RouterLink } from '@angular/router';

import { RedocService } from '../../../../services/redoc.service';
import { ApiInfo, specPath } from '../../../../utils/apis';

type Status = 'loading' | 'ready' | 'failed';

/**
 * Redoc styles itself and has no dark mode, so its panel always uses Redoc's light theme,
 * tinted with the site's primary colour.
 */
const REDOC_THEME = {
  colors: { primary: { main: '#a43c00' } },
  typography: {
    fontFamily: 'Roboto, system-ui, sans-serif',
    headings: { fontFamily: 'Roboto, system-ui, sans-serif', fontWeight: '500' },
    code: { fontFamily: '"Roboto Mono", ui-monospace, monospace' },
  },
  sidebar: { activeTextColor: '#a43c00' },
  rightPanel: { backgroundColor: '#14161d' },
};

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
    <div #host class="redoc" [hidden]="status() !== 'ready'"></div>
  `,
})
export class RedocView {
  private readonly redoc = inject(RedocService);
  private readonly host = viewChild.required<ElementRef<HTMLElement>>('host');

  readonly api = input.required<ApiInfo>();
  /** Distance from the top of the viewport to the bottom of whatever is pinned above Redoc. */
  readonly scrollOffset = input<() => number>(() => 0);

  protected readonly status = signal<Status>('loading');
  private run = 0;

  constructor() {
    afterRenderEffect({
      write: () => {
        const api = this.api();
        const host = this.host().nativeElement;
        untracked(() => this.render(api, host));
      },
    });
  }

  private async render(api: ApiInfo, host: HTMLElement): Promise<void> {
    // Ignore a slow render that finishes after the user moved on to another API.
    const run = ++this.run;
    this.status.set('loading');
    try {
      await this.redoc.render(specPath(api), host, {
        scrollYOffset: this.scrollOffset(),
        hideDownloadButtons: true,
        expandResponses: '200,201',
        theme: REDOC_THEME,
      });
      if (run === this.run) this.status.set('ready');
    } catch (error) {
      console.error(error);
      if (run === this.run) this.status.set('failed');
    }
  }
}
