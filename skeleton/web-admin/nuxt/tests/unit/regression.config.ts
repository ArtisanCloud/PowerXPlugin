import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import { fileURLToPath } from 'node:url';
const path = (value: string) => fileURLToPath(new URL(value, import.meta.url));
export default defineConfig({plugins:[vue()],test:{environment:'node',include:['tests/unit/regression-workbench.spec.ts']},resolve:{alias:{'#imports':path('./helpers/regression-imports.ts')}}});
