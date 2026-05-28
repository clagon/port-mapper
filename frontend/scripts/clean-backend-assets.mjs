import { rm, mkdir, readdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const staticDir = resolve(here, '../../backend/assets/static');

await mkdir(staticDir, { recursive: true });

for (const entry of await readdir(staticDir)) {
  if (entry === '.gitkeep') {
    continue;
  }
  await rm(resolve(staticDir, entry), { recursive: true, force: true });
}
