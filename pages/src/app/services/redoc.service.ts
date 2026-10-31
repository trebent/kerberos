import { DOCUMENT, inject, Service } from '@angular/core';

/** The subset of Redoc's options this site uses. See https://redocly.com/docs/redoc/config */
export interface RedocOptions {
  scrollYOffset?: number | string | (() => number);
  hideDownloadButtons?: boolean;
  expandResponses?: string;
  theme?: Record<string, unknown>;
}

interface RedocGlobal {
  init(
    specUrl: string,
    options: RedocOptions,
    element: HTMLElement,
    onLoaded?: (error?: unknown) => void,
  ): void;
}

/** Copied out of node_modules by the `assets` entry in angular.json; relative to <base href>. */
const SCRIPT_PATH = 'redoc/redoc.standalone.js';

/**
 * Renders OpenAPI specs with Redoc. The standalone bundle is large (React and a YAML parser), so
 * it is downloaded the first time a reference is shown instead of being part of the app bundle.
 */
@Service()
export class RedocService {
  private readonly document = inject(DOCUMENT);
  private loading?: Promise<RedocGlobal>;

  /**
   * Renders the spec at `specUrl` (relative to <base href>) into a fresh element in `host`,
   * replacing its content.
   */
  async render(specUrl: string, host: HTMLElement, options: RedocOptions): Promise<void> {
    const redoc = await this.load();
    // Redoc resolves relative URLs against the page URL, not <base href>.
    const absoluteUrl = new URL(specUrl, this.document.baseURI).href;
    const element = this.document.createElement('div');
    host.replaceChildren(element);

    await new Promise<void>((resolve, reject) =>
      redoc.init(absoluteUrl, options, element, (error) => (error ? reject(error) : resolve())),
    );
  }

  private load(): Promise<RedocGlobal> {
    this.loading ??= new Promise<RedocGlobal>((resolve, reject) => {
      const window = this.document.defaultView as (Window & { Redoc?: RedocGlobal }) | null;
      if (window?.Redoc) {
        resolve(window.Redoc);
        return;
      }

      const script = this.document.createElement('script');
      script.src = SCRIPT_PATH;
      script.async = true;
      script.onload = () =>
        window?.Redoc ? resolve(window.Redoc) : reject(new Error('Redoc did not register itself'));
      script.onerror = () => {
        // Allow a retry on the next render, e.g. after a network blip.
        this.loading = undefined;
        script.remove();
        reject(new Error(`Could not load ${SCRIPT_PATH}`));
      };
      this.document.head.appendChild(script);
    });
    return this.loading;
  }
}
