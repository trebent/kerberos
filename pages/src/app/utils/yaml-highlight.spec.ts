import { highlightYamlLine, YamlToken } from './yaml-highlight';

const kinds = (tokens: YamlToken[]) => tokens.map((t) => `${t.kind}:${t.text}`);

describe('highlightYamlLine', () => {
  it('colours comments', () => {
    expect(kinds(highlightYamlLine('  # note'))).toEqual(['plain:  ', 'comment:# note']);
  });

  it('colours keys and scalar values', () => {
    expect(kinds(highlightYamlLine('openapi: "3.0.0"'))).toEqual([
      'key:openapi',
      'plain::',
      'plain: ',
      'string:"3.0.0"',
    ]);
    expect(kinds(highlightYamlLine('  minLength: 1'))).toEqual([
      'plain:  ',
      'key:minLength',
      'plain::',
      'plain: ',
      'number:1',
    ]);
  });

  it('handles list items and leaves everything else plain', () => {
    expect(kinds(highlightYamlLine('  - name: id'))).toEqual([
      'plain:  - ',
      'key:name',
      'plain::',
      'plain: id',
    ]);
    expect(kinds(highlightYamlLine('  - "a"'))).toEqual(['plain:  -', 'plain: ', 'string:"a"']);
    expect(highlightYamlLine('').length).toBe(0);
  });

  it('keeps every character of the line', () => {
    const line = "    $ref: '#/components/schemas/User' # trailing";
    expect(
      highlightYamlLine(line)
        .map((t) => t.text)
        .join(''),
    ).toBe(line);
  });
});
