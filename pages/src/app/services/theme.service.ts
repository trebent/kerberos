import { computed, DOCUMENT, effect, inject, Service, Signal, signal } from '@angular/core';

export type ThemeMode = 'system' | 'light' | 'dark';

/** The colours actually shown: `mode`, with `system` resolved. */
export type ColorScheme = 'light' | 'dark';

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

  private readonly systemDark = this.watchSystemDark();

  /** For content that cannot use `light-dark()`, such as Redoc. */
  readonly scheme = computed<ColorScheme>(() => {
    const mode = this.mode();
    if (mode !== 'system') return mode;
    return this.systemDark() ? 'dark' : 'light';
  });

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

  private watchSystemDark(): Signal<boolean> {
    const query = this.document.defaultView?.matchMedia?.('(prefers-color-scheme: dark)');
    const dark = signal(query?.matches ?? false);
    query?.addEventListener('change', (event) => dark.set(event.matches));
    return dark.asReadonly();
  }
}
