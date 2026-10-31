import { CanMatchFn } from '@angular/router';

import { findApi } from '../utils/apis';

/** Only matches `api/:api` for an API in the catalogue; anything else falls through to `**`. */
export const knownApiGuard: CanMatchFn = (_route, segments) => !!findApi(segments[1]?.path);
