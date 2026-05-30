import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { docsPages, generatedDocsDir, generatedRoot } from './docs-data.mjs';

function walkMarkdown(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walkMarkdown(path));
    } else if (entry.isFile() && entry.name.endsWith('.md')) {
      files.push(path);
    }
  }
  return files;
}

function routeForGeneratedFile(path) {
  const rel = relative(generatedRoot, path).replace(/\\/g, '/').replace(/\.md$/, '');
  return rel === 'index' ? '' : rel.replace(/\/index$/, '');
}

function isExternalUrl(url) {
  return /^(?:[a-z][a-z0-9+.-]*:|\/\/)/i.test(url);
}

function resolveRoute(fromRoute, href) {
  const [pathPart] = href.split(/[?#]/);
  if (!pathPart || href.startsWith('#') || isExternalUrl(href)) {
    return undefined;
  }
  if (pathPart.startsWith('/terraformer/')) {
    throw new Error('Generated link should be relative, not base-prefixed: ' + href);
  }
  if (pathPart.startsWith('/')) {
    throw new Error('Generated link should be relative, not root-relative: ' + href);
  }

  const base = fromRoute ? fromRoute.split('/') : [];
  const parts = [...base];
  for (const segment of pathPart.split('/')) {
    if (!segment || segment === '.') {
      continue;
    }
    if (segment === '..') {
      parts.pop();
      continue;
    }
    parts.push(segment);
  }
  return parts.join('/');
}

function main() {
  const requiredFiles = [
    join(generatedRoot, 'index.md'),
    join(generatedDocsDir, 'index.md'),
    join(generatedDocsDir, 'install.md'),
    join(generatedDocsDir, 'providers.md'),
    ...docsPages().map((page) => join(generatedDocsDir, page.slug + '.md')),
  ];

  for (const file of requiredFiles) {
    if (!existsSync(file)) {
      throw new Error('Missing generated route file: ' + file);
    }
  }

  const markdownFiles = walkMarkdown(generatedRoot);
  const routes = new Set(markdownFiles.map(routeForGeneratedFile));
  const errors = [];
  const linkPattern = /!?\[[^\]]+\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;

  for (const file of markdownFiles) {
    const fromRoute = routeForGeneratedFile(file);
    const markdown = readFileSync(file, 'utf8');
    for (const match of markdown.matchAll(linkPattern)) {
      const full = match[0];
      if (full.startsWith('![')) {
        continue;
      }
      try {
        const targetRoute = resolveRoute(fromRoute, match[1]);
        if (targetRoute !== undefined && !routes.has(targetRoute)) {
          errors.push(relative(process.cwd(), file) + ' links to missing route ' + match[1]);
        }
      } catch (error) {
        errors.push(relative(process.cwd(), file) + ': ' + error.message);
      }
    }
  }

  if (errors.length > 0) {
    throw new Error(errors.join('\n'));
  }

  console.log('Validated ' + markdownFiles.length + ' generated Markdown files.');
}

main();
