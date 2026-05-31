// NOTE: This is a narrow regression guard for CLA-26.
// It exists until the frontend has a broader component or integration test suite.
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import assert from 'node:assert/strict';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const app = readFileSync(resolve(root, 'src/App.svelte'), 'utf8');
const dashboard = readFileSync(resolve(root, 'src/Dashboard.svelte'), 'utf8');

const refreshBody = app.match(/async function refresh\(\) \{([\s\S]*?)\n[ ]{2}\}/)?.[1];
assert(refreshBody, 'App.svelte must define refresh()');
assert(!refreshBody.includes('api.discover'), 'refresh() must not call api.discover()');

const discoverBody = app.match(/async function discover\(\) \{([\s\S]*?)\n[ ]{2}\}/)?.[1];
assert(discoverBody, 'App.svelte must define a separate discover() action');
assert(discoverBody.includes('api.discover'), 'discover() must call api.discover()');

assert(app.includes('discover={discover}'), 'Dashboard must receive the discover action');
assert(dashboard.includes('export let discover'), 'Dashboard must expose a discover prop');
assert(dashboard.includes('on:click={discover}'), 'Dashboard must provide an explicit rediscovery button');
