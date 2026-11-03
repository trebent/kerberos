import { HttpClient } from '@angular/common/http';
import { DOCUMENT, inject, Service } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ColorScheme } from './theme.service';

type RedocModule = typeof import('redoc/bundles/redoc.standalone.js');

/** The options Redoc's `init()` accepts. See https://redocly.com/docs/redoc/v3.x/config */
export type RedocOptions = NonNullable<Parameters<RedocModule['init']>[1]>;

/** A rendered reference. Call `destroy()` before its element is reused or removed. */
export interface RedocHandle {
  destroy(): void;
}

/** Redoc keeps its own light/dark choice here and in a class on <html>. */
const COLOR_MODE_KEY = 'redoc.colorMode';
const COLOR_MODES: readonly ColorScheme[] = ['light', 'dark'];

/**
 * Renders OpenAPI specs with Redoc. The standalone bundle is large (React and a YAML parser), so
 * it is a lazy chunk, downloaded the first time a reference is shown.
 */
@Service()
export class RedocService {
  private readonly document = inject(DOCUMENT);
  private readonly http = inject(HttpClient);
  private loading?: Promise<RedocModule>;

  /**
   * Fetches the spec at `specUrl` (relative to <base href>) and renders it into `element`.
   * Rejects if Redoc or the spec cannot be downloaded.
   */
  async render(specUrl: string, element: HTMLElement, options: RedocOptions): Promise<RedocHandle> {
    // Fetched here rather than by Redoc, whose init() does not report download errors.
    const [redoc, spec] = await Promise.all([
      this.load(),
      firstValueFrom(this.http.get(specUrl, { responseType: 'text' })),
    ]);
    const root = redoc.init(spec, options, element);
    return { destroy: () => root.unmount() };
  }

  /** Switches Redoc between its light and dark themes, including references rendered later. */
  setColorMode(scheme: ColorScheme): void {
    const root = this.document.documentElement;
    root.classList.remove(...COLOR_MODES);
    root.classList.add(scheme);
    try {
      localStorage.setItem(COLOR_MODE_KEY, scheme);
    } catch {
      // Storage unavailable: init() falls back to the class set above.
    }
  }

  private load(): Promise<RedocModule> {
    this.loading ??= import('redoc/bundles/redoc.standalone.js').catch((error: unknown) => {
      // Allow a retry on the next render, e.g. after a network blip.
      this.loading = undefined;
      throw error;
    });
    return this.loading;
  }
}
