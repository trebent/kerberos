export type YamlTokenKind = 'comment' | 'key' | 'string' | 'number' | 'plain';

export interface YamlToken {
  kind: YamlTokenKind;
  text: string;
}

const COMMENT = /^(\s*)(#.*)$/;
const KEY = /^(\s*(?:-\s+)?)([A-Za-z0-9_$/{}.\-"']+)(:)(?=\s|$)/;
const VALUE = /^(\s+)("[^"]*"|'[^']*'|-?\d+(?:\.\d+)?)\s*$/;

/**
 * Splits one line of YAML into tokens for the source view. Deliberately small: it colours
 * comments, mapping keys and scalar values, which is all an OpenAPI file needs to be readable.
 */
export function highlightYamlLine(line: string): YamlToken[] {
  const comment = COMMENT.exec(line);
  if (comment) {
    return compact([
      { kind: 'plain', text: comment[1] },
      { kind: 'comment', text: comment[2] },
    ]);
  }

  const tokens: YamlToken[] = [];
  let rest = line;

  const key = KEY.exec(rest);
  if (key) {
    tokens.push({ kind: 'plain', text: key[1] }, { kind: 'key', text: key[2] });
    tokens.push({ kind: 'plain', text: key[3] });
    rest = rest.slice(key[0].length);
  } else {
    const item = /^(\s*-)(?=\s)/.exec(rest);
    if (item) {
      tokens.push({ kind: 'plain', text: item[1] });
      rest = rest.slice(item[0].length);
    }
  }

  const value = VALUE.exec(rest);
  if (value && (key || tokens.length > 0)) {
    const kind: YamlTokenKind = /^["']/.test(value[2]) ? 'string' : 'number';
    tokens.push({ kind: 'plain', text: value[1] }, { kind, text: value[2] });
    rest = rest.slice(value[1].length + value[2].length);
  }

  tokens.push({ kind: 'plain', text: rest });
  return compact(tokens);
}

function compact(tokens: YamlToken[]): YamlToken[] {
  return tokens.filter((token) => token.text !== '');
}
