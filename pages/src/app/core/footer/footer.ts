import { Component } from '@angular/core';

import {
  DocLink,
  docsUrl,
  ISSUES_URL,
  LICENSE_URL,
  RELEASES_URL,
  REPO_URL,
} from '../../utils/links';

@Component({
  selector: 'app-footer',
  styleUrl: './footer.scss',
  templateUrl: './footer.html',
})
export class Footer {
  protected readonly year = new Date().getFullYear();
  protected readonly license = LICENSE_URL;

  protected readonly docs: readonly DocLink[] = [
    { label: 'Documentation index', href: docsUrl() },
    { label: 'Getting started', href: docsUrl('getting-started.md') },
    { label: 'Configuration', href: docsUrl('configuration.md') },
    { label: 'Deployment', href: docsUrl('deployment.md') },
  ];

  protected readonly project: readonly DocLink[] = [
    { label: 'Source on GitHub', href: REPO_URL },
    { label: 'Releases', href: RELEASES_URL },
    { label: 'Report an issue', href: ISSUES_URL },
  ];
}
