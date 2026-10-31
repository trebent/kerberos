import { httpResource } from '@angular/common/http';
import { Component, computed, input } from '@angular/core';

import { CopyButton } from '../../../../components/copy-button/copy-button';
import { ApiInfo, specPath } from '../../../../utils/apis';
import { highlightYamlLine } from '../../../../utils/yaml-highlight';

/** The raw OpenAPI YAML, with line numbers and light syntax colouring. */
@Component({
  imports: [CopyButton],
  selector: 'app-spec-source',
  styleUrl: './spec-source.scss',
  templateUrl: './spec-source.html',
})
export class SpecSource {
  readonly api = input.required<ApiInfo>();

  protected readonly spec = httpResource.text(() => specPath(this.api()));
  protected readonly lines = computed(() =>
    (this.spec.hasValue() ? this.spec.value() : '')
      .replace(/\n$/, '')
      .split('\n')
      .map(highlightYamlLine),
  );
}
