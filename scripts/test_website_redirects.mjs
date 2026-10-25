import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

const html = readFileSync(new URL('../web/index.html', import.meta.url), 'utf8');
const startup = html.match(/<script>\s*([\s\S]*?)<\/script>/)?.[1];
assert.ok(startup, 'Website startup script is missing');
assert.match(html, /<link rel="canonical" href="https:\/\/domain-check\.songming\.org\/">/);

const cases = [
  ['https://domain-check.songming.org/index.html', '/'],
  ['https://domain-check.songming.org/index', '/'],
  ['https://domain-check.songming.org/index.html?source=github#install', '/?source=github#install'],
  ['https://domain-check.songming.org/index?next=https%3A%2F%2Fexample.test#why', '/?next=https%3A%2F%2Fexample.test#why'],
  ['http://domain-check.songming.org/index', '/'],
  ['http://127.0.0.1:20772/index.html?preview=1#install', '/?preview=1#install'],
  ['https://domain-check.songming.org/', null],
  ['https://domain-check.songming.org/?page=index.html#install', null],
  ['https://domain-check.songming.org/index.html.bak', null],
  ['https://domain-check.songming.org/other/index.html', null],
  ['file:///local/domain-check/web/index.html', null],
  ['file:///local/domain-check/web/index.html?preview=1#install', null],
];

for (const [address, expected] of cases) {
  const url = new URL(address);
  const redirects = [];
  const element = { dataset: {} };
  vm.runInNewContext(startup, {
    location: {
      protocol: url.protocol, pathname: url.pathname, search: url.search, hash: url.hash,
      replace: target => redirects.push(target),
    },
    navigator: { language: 'en-US' },
    localStorage: { getItem: () => null },
    document: { documentElement: element },
  }, { timeout: 1000 });
  assert.deepEqual(redirects, expected === null ? [] : [expected], address);
  if (expected !== null) {
    const target = new URL(redirects[0], url);
    assert.equal(target.origin, url.origin, 'Redirect must remain on the same origin');
    assert.equal(target.pathname, '/');
    assert.equal(element.lang, undefined, 'Redirect must run before page initialization');
  } else {
    assert.equal(element.lang, 'en', 'Normal and local-file pages must initialize');
  }
}
console.log(`Website redirect checks passed (${cases.length} cases).`);
