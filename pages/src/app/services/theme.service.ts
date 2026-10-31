import { DOCUMENT, effect, inject, Service, signal } from '@angular/core';

export type ThemeMode = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'kerberos.theme';
const MODES: readonly ThemeMode[] = ['system', 'light', 'dark'];

function storedMode(): ThemeMode {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    return MODES.find((mode) => mode === saved) ?? 'system';
  } catch {
    return 'system';
  }
}

/**
 * Light/dark preference. The Material theme in `styles.scss` is built on `light-dark()`, so
 * switching is just a matter of setting `color-scheme` on the root element.
 */
@Service()
export class ThemeService {
  private readonly document = inject(DOCUMENT);

  readonly mode = signal<ThemeMode>(storedMode());

  constructor() {
    effect(() => {
      const mode = this.mode();
      this.document.documentElement.style.colorScheme = mode === 'system' ? 'light dark' : mode;

      try {
        localStorage.setItem(STORAGE_KEY, mode);
      } catch {
        // Storage unavailable: the choice lasts for this page load.
      }
    });
  }
}
