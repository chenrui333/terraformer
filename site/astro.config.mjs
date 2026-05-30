import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { getSidebar } from './scripts/docs-data.mjs';

export default defineConfig({
  site: 'https://chenrui333.github.io',
  base: '/terraformer',
  integrations: [
    starlight({
      title: 'Terraformer',
      description:
        'Import existing infrastructure into Terraform configuration and state.',
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/chenrui333/terraformer',
        },
      ],
      customCss: ['./src/styles/custom.css'],
      sidebar: getSidebar(),
    }),
  ],
});
