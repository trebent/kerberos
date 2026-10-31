import { Component } from '@angular/core';
import { MatIcon } from '@angular/material/icon';

import { docsUrl } from '../../../../utils/links';

interface Stage {
  name: string;
  icon: string;
  does: string;
  /** Status codes this stage answers with when it stops the request. */
  rejects: readonly string[];
  optional?: boolean;
  doc: string;
}

/**
 * How the gateway handles one call to `/gw/backend/<name>/<path>`. Kept in step with
 * "Request lifecycle" in docs/routing.md, which is the source of truth.
 */
const STAGES: readonly Stage[] = [
  {
    name: 'Observe',
    icon: 'monitoring',
    does: 'Starts a trace span, metrics and logging, and reads the backend name from the path.',
    rejects: ['400'],
    doc: docsUrl('observability.md'),
  },
  {
    name: 'Route',
    icon: 'alt_route',
    does: 'Finds the backend, applies its CORS policy and strips the /gw/backend/<name> prefix.',
    rejects: ['404', '403'],
    doc: docsUrl('routing.md'),
  },
  {
    name: 'Authenticate',
    icon: 'lock',
    does: 'Checks the session cookie and group rules, then adds X-Krb-User, X-Krb-Org and X-Krb-Groups.',
    rejects: ['401', '403'],
    optional: true,
    doc: docsUrl('authentication.md'),
  },
  {
    name: 'Validate',
    icon: 'rule',
    does: "Checks the path, parameters and body against the backend's OpenAPI spec.",
    rejects: ['400', '404'],
    optional: true,
    doc: docsUrl('request-validation.md'),
  },
  {
    name: 'Forward',
    icon: 'send',
    does: 'Sends the request to the backend and relays its response to the client.',
    rejects: [],
    doc: docsUrl('routing.md#what-is-forwarded'),
  },
];

@Component({
  imports: [MatIcon],
  selector: 'app-request-flow',
  styleUrl: './request-flow.scss',
  templateUrl: './request-flow.html',
})
export class RequestFlow {
  protected readonly stages = STAGES;
  protected readonly lifecycleDoc = docsUrl('routing.md#request-lifecycle');
  protected readonly orderDoc = docsUrl('routing.md#pipeline-order');
}
