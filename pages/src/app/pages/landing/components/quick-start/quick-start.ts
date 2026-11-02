import { Component } from '@angular/core';

import { CopyButton } from '../../../../components/copy-button/copy-button';

interface TerminalLine {
  kind: 'command' | 'comment' | 'output' | 'blank';
  text: string;
}

const LINES: readonly TerminalLine[] = [
  { kind: 'comment', text: '# scaffold a Compose stack with an echo test backend' },
  { kind: 'command', text: 'krbctl compose -y --echo' },
  { kind: 'command', text: 'krbctl config -y --echo' },
  { kind: 'command', text: 'docker compose up -d' },
  { kind: 'blank', text: '' },
  { kind: 'comment', text: '# send a request through the gateway' },
  { kind: 'command', text: 'curl -i localhost:30000/gw/backend/echo/hello' },
  { kind: 'output', text: 'HTTP/1.1 200 OK' },
];

/** The four commands from the getting-started guide, in a terminal-style panel. */
@Component({
  imports: [CopyButton],
  selector: 'app-quick-start',
  styleUrl: './quick-start.scss',
  templateUrl: './quick-start.html',
})
export class QuickStart {
  protected readonly lines = LINES;
  protected readonly commands = LINES.filter((line) => line.kind === 'command')
    .map((line) => line.text)
    .join('\n');

  protected tokenClass(kind: TerminalLine['kind']): string {
    return kind === 'comment' ? 'tok-comment' : kind === 'output' ? 'ok' : '';
  }
}
