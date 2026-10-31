import { Component, computed, inject } from '@angular/core';
import { MatButton, MatIconButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { MatMenu, MatMenuItem, MatMenuTrigger } from '@angular/material/menu';
import { MatToolbar } from '@angular/material/toolbar';
import { RouterLink, RouterLinkActive } from '@angular/router';

import { ThemeMode, ThemeService } from '../../services/theme.service';
import { docsUrl, REPO_URL } from '../../utils/links';

interface ThemeOption {
  mode: ThemeMode;
  label: string;
  icon: string;
}

@Component({
  imports: [
    MatButton,
    MatIcon,
    MatIconButton,
    MatMenu,
    MatMenuItem,
    MatMenuTrigger,
    MatToolbar,
    RouterLink,
    RouterLinkActive,
  ],
  selector: 'app-header',
  styleUrl: './header.scss',
  templateUrl: './header.html',
})
export class Header {
  protected readonly theme = inject(ThemeService);
  protected readonly docs = docsUrl();
  protected readonly repo = REPO_URL;

  protected readonly themeOptions: readonly ThemeOption[] = [
    { mode: 'system', label: 'System', icon: 'brightness_auto' },
    { mode: 'light', label: 'Light', icon: 'light_mode' },
    { mode: 'dark', label: 'Dark', icon: 'dark_mode' },
  ];

  protected readonly currentTheme = computed(
    () =>
      this.themeOptions.find((option) => option.mode === this.theme.mode()) ?? this.themeOptions[0],
  );
}
