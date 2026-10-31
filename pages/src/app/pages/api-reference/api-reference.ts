import { Component, computed, ElementRef, inject, input, viewChild } from '@angular/core';
import { MatButton } from '@angular/material/button';
import { MatButtonToggle, MatButtonToggleGroup } from '@angular/material/button-toggle';
import { MatIcon } from '@angular/material/icon';
import { MatTabLink, MatTabNav, MatTabNavPanel } from '@angular/material/tabs';
import { Router, RouterLink } from '@angular/router';

import { APIS, findApi, specPath } from '../../utils/apis';
import { repoFileUrl } from '../../utils/links';
import { RedocView } from './components/redoc-view/redoc-view';
import { SpecSource } from './components/spec-source/spec-source';

export type ApiView = 'reference' | 'source';

/**
 * `/api/:api?view=source`: one OpenAPI contract, rendered by Redoc or shown as YAML.
 * Both `api` (route parameter) and `view` (query parameter) are bound by the router.
 */
@Component({
  imports: [
    MatButton,
    MatButtonToggle,
    MatButtonToggleGroup,
    MatIcon,
    MatTabLink,
    MatTabNav,
    MatTabNavPanel,
    RedocView,
    RouterLink,
    SpecSource,
  ],
  selector: 'app-api-reference',
  styleUrl: './api-reference.scss',
  templateUrl: './api-reference.html',
})
export class ApiReference {
  private readonly router = inject(Router);
  private readonly bar = viewChild.required<ElementRef<HTMLElement>>('bar');

  readonly api = input.required<string>();
  readonly view = input<string>();

  protected readonly apis = APIS;
  // knownApiGuard only lets catalogue IDs through; the fallback keeps the types simple.
  protected readonly current = computed(() => findApi(this.api()) ?? APIS[0]);
  protected readonly currentView = computed<ApiView>(() =>
    this.view() === 'source' ? 'source' : 'reference',
  );
  protected readonly specUrl = computed(() => specPath(this.current()));
  protected readonly githubUrl = computed(() => repoFileUrl(specPath(this.current())));

  /** Redoc keeps its sidebar and anchors clear of the sticky header and API bar. */
  protected readonly scrollOffset = (): number =>
    this.bar().nativeElement.getBoundingClientRect().bottom;

  protected setView(view: ApiView): void {
    void this.router.navigate([], {
      queryParams: { view: view === 'source' ? 'source' : null },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });
  }
}
