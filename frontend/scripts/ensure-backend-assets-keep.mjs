import { mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const staticDir = resolve(here, '../../backend/assets/static');

await mkdir(staticDir, { recursive: true });
await writeFile(resolve(staticDir, '.gitkeep'), '');
