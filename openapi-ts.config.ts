import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: './spec/swagger.json',
  output: 'packages/sdk/src',
  plugins: [
    '@hey-api/typescript',
    {
      name: '@hey-api/transformers',
      dates: true,
      bigInt: true,
    },
    {
      name: '@hey-api/sdk',
      transformer: true
    },
    '@hey-api/client-fetch',
    '@pinia/colada',
  ],
})