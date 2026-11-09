import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { unified } from '@astrojs/markdown-remark';
import { getSidebar } from './scripts/docs-data.mjs';

export default defineConfig({
  site: 'https://chenrui333.github.io',
  base: '/terraformer',
  // Retain the Astro 6 Markdown pipeline and inline whitespace behavior.
  markdown: { processor: unified() },
  compressHTML: true,
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
