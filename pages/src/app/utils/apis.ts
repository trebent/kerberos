import { DocLink, docsUrl } from './links';

export type ApiId = 'gateway' | 'admin' | 'auth';

/** One of the HTTP APIs Kerberos serves, described by a contract in the repository's `openapi/`. */
export interface ApiInfo {
  id: ApiId;
  /** Full name, used in headings and the page title. */
  name: string;
  /** Short name, used in tabs. */
  shortName: string;
  /** File name under `openapi/`. */
  file: string;
  /** Default port and path prefix, e.g. `:30001 · /api/admin`. */
  endpoint: string;
  summary: string;
  /** The pages in `docs/` that explain this API. */
  guides: readonly DocLink[];
}

export const APIS: readonly ApiInfo[] = [
  {
    id: 'gateway',
    name: 'Gateway API',
    shortName: 'Gateway',
    file: 'gateway.yaml',
    endpoint: ':30000 · /gw',
    summary: 'Proxied backend traffic and the health endpoint.',
    guides: [
      { label: 'Routing', href: docsUrl('routing.md') },
      { label: 'Authentication', href: docsUrl('authentication.md') },
    ],
  },
  {
    id: 'admin',
    name: 'Admin API',
    shortName: 'Admin',
    file: 'admin.yaml',
    endpoint: ':30001 · /api/admin',
    summary: 'Superuser, admin users and groups, permissions, flow inspection and debug sessions.',
    guides: [
      { label: 'Admin API', href: docsUrl('admin-api.md') },
      { label: 'Debug sessions', href: docsUrl('admin-debugging.md') },
    ],
  },
  {
    id: 'auth',
    name: 'Basic-auth API',
    shortName: 'Basic auth',
    file: 'auth_basic.yaml',
    endpoint: ':30001 · /api/auth/basic',
    summary: 'Organisations, users, groups, and the login, refresh and logout flow.',
    guides: [
      { label: 'Organisations', href: docsUrl('organizations.md') },
      { label: 'Sessions', href: docsUrl('sessions.md') },
    ],
  },
];

export function findApi(id: string | null | undefined): ApiInfo | undefined {
  return APIS.find((api) => api.id === id);
}

/** Relative to `<base href>`: the build copies `../openapi/*.yaml` into `openapi/`. */
export function specPath(api: ApiInfo): string {
  return `openapi/${api.file}`;
}
