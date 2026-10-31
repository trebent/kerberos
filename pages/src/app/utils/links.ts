/** Where the source, docs and releases live. Every external link on the site is built here. */
export const REPO_URL = 'https://github.com/trebent/kerberos';

/** A page in the official documentation under `docs/`, e.g. `docsUrl('routing.md')`. */
export function docsUrl(page = 'README.md'): string {
  return `${REPO_URL}/blob/main/docs/${page}`;
}

/** A file in the repository on the main branch, e.g. `repoFileUrl('openapi/admin.yaml')`. */
export function repoFileUrl(path: string): string {
  return `${REPO_URL}/blob/main/${path}`;
}

export const RELEASES_URL = `${REPO_URL}/releases`;
export const ISSUES_URL = `${REPO_URL}/issues`;
export const LICENSE_URL = `${REPO_URL}/blob/main/LICENSE`;

export interface DocLink {
  label: string;
  href: string;
}
