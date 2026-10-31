import { Component } from '@angular/core';
import { MatButton } from '@angular/material/button';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';

import { APIS } from '../../utils/apis';
import { DocLink, docsUrl } from '../../utils/links';
import { QuickStart } from './components/quick-start/quick-start';
import { RequestFlow } from './components/request-flow/request-flow';

interface DocCard extends DocLink {
  icon: string;
  summary: string;
}

@Component({
  imports: [MatButton, MatIcon, QuickStart, RequestFlow, RouterLink],
  selector: 'app-landing',
  styleUrl: './landing.scss',
  templateUrl: './landing.html',
})
export class Landing {
  protected readonly apis = APIS;
  protected readonly docsIndex = docsUrl();
  protected readonly gettingStarted = docsUrl('getting-started.md');

  /** A handful of entry points; the docs index lists everything else. */
  protected readonly docs: readonly DocCard[] = [
    {
      icon: 'rocket_launch',
      label: 'Getting started',
      href: docsUrl('getting-started.md'),
      summary: 'From an empty directory to a logged-in user calling a protected backend.',
    },
    {
      icon: 'deployed_code',
      label: 'Deployment',
      href: docsUrl('deployment.md'),
      summary: 'Images, environment variables, health checks and a production checklist.',
    },
    {
      icon: 'tune',
      label: 'Configuration',
      href: docsUrl('configuration.md'),
      summary: 'The JSON config file, its references and where each section is documented.',
    },
    {
      icon: 'lock',
      label: 'Authentication',
      href: docsUrl('authentication.md'),
      summary: 'Protect backends with logins and groups, and read the identity headers.',
    },
  ];
}
