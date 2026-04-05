import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      // Stub @vue-flow/node-resizer in tests until npm install is run
      '@vue-flow/node-resizer': resolve(__dirname, 'src/test-stubs/vue-flow-node-resizer.ts'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
  },
})
