import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { basename, join, posix } from 'node:path';
import { fileURLToPath } from 'node:url';

const siteDir = fileURLToPath(new URL('..', import.meta.url));
export const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
export const readmePath = join(repoRoot, 'README.md');
export const docsDir = join(repoRoot, 'docs');
export const generatedRoot = join(siteDir, 'src/content/docs');
export const generatedDocsDir = join(generatedRoot, 'docs');

const unsafeSlugPattern = /[^a-z0-9-]/;
const backtick = String.fromCharCode(96);
const fenceMarker = backtick.repeat(3);
const inlineCodePattern = new RegExp(backtick + '([^' + backtick + ']+)' + backtick, 'g');

function fenceBoundary(line) {
  const match = new RegExp('^(' + backtick + '{3,}|~{3,})').exec(line.trim());
  if (!match) {
    return undefined;
  }
  return {
    marker: match[1][0],
    length: match[1].length,
  };
}

function matchingFence(line, fence) {
  const boundary = fenceBoundary(line);
  return boundary && boundary.marker === fence.marker && boundary.length >= fence.length;
}

function firstMarkdownHeading(markdown, maxDepth) {
  let fence;
  const lines = markdown.split(/\r?\n/);

  for (const [index, line] of lines.entries()) {
    if (fence) {
      if (matchingFence(line, fence)) {
        fence = undefined;
      }
      continue;
    }

    const boundary = fenceBoundary(line);
    if (boundary) {
      fence = boundary;
      continue;
    }

    const match = /^(#{1,6})\s+(.+)$/.exec(line);
    if (match && match[1].length <= maxDepth) {
      return {
        index,
        title: stripMarkdown(match[2]),
      };
    }
  }

  return undefined;
}

export function readUtf8(path) {
  return readFileSync(path, 'utf8');
}

export function normalizeSlug(fileName) {
  const slug = basename(fileName, '.md').toLowerCase();
  if (!slug || /\s/.test(slug) || unsafeSlugPattern.test(slug)) {
    throw new Error('Unsafe docs slug derived from ' + fileName + ': ' + slug);
  }
  return slug;
}

export function parseProviderCategories(readme = readUtf8(readmePath)) {
  const categories = [];
  let currentCategory;
  let inProviders = false;

  for (const line of readme.split(/\r?\n/)) {
    if (line === '- [Supported Providers](/docs)') {
      inProviders = true;
      continue;
    }

    if (!inProviders) {
      continue;
    }

    if (/^- \[/.test(line)) {
      break;
    }

    const categoryMatch = /^    \* (.+)$/.exec(line);
    if (categoryMatch) {
      currentCategory = { label: categoryMatch[1], items: [] };
      categories.push(currentCategory);
      continue;
    }

    const providerMatch = /^        \* \[([^\]]+)\]\((\/docs\/[^)]+\.md)\)$/.exec(line);
    if (providerMatch) {
      if (!currentCategory) {
        throw new Error('Provider entry has no category: ' + line);
      }
      const sourcePath = providerMatch[2].replace(/^\//, '');
      currentCategory.items.push({
        label: providerMatch[1],
        sourcePath,
        slug: normalizeSlug(sourcePath),
      });
    }
  }

  if (categories.length === 0) {
    throw new Error('Could not find Supported Providers list in README.md');
  }

  return categories;
}

export function providerLabelMap(categories = parseProviderCategories()) {
  const labels = new Map();
  for (const category of categories) {
    for (const item of category.items) {
      labels.set(item.slug, item.label);
    }
  }
  return labels;
}

export function docsFiles() {
  return readdirSync(docsDir)
    .filter((file) => file.endsWith('.md'))
    .sort((a, b) => a.localeCompare(b));
}

export function stripMarkdown(value) {
  return value
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(inlineCodePattern, '$1')
    .replace(/[*_~]/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

export function firstHeading(markdown) {
  return firstMarkdownHeading(markdown, 6)?.title;
}

export function removeFirstH1(markdown) {
  const lines = markdown.split(/\r?\n/);
  const heading = firstMarkdownHeading(markdown, 1);
  if (!heading) {
    return markdown;
  }
  lines.splice(heading.index, 1);
  return lines.join('\n').replace(/^\n+/, '');
}

export function firstParagraph(markdown) {
  let inFence = false;
  const paragraph = [];

  for (const line of markdown.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (trimmed.startsWith(fenceMarker)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) {
      continue;
    }
    if (!trimmed) {
      if (paragraph.length > 0) {
        break;
      }
      continue;
    }
    if (/^(#{1,6}|[-*+]\s|\d+\.\s|>)/.test(trimmed)) {
      if (paragraph.length > 0) {
        break;
      }
      continue;
    }
    paragraph.push(trimmed);
  }

  return stripMarkdown(paragraph.join(' '));
}

export function shortDescription(markdown, fallback) {
  const paragraph = firstParagraph(markdown);
  const value = paragraph || fallback;
  return value.length > 156 ? value.slice(0, 153).trim() + '...' : value;
}

export function titleFromSlug(slug) {
  return slug
    .split('-')
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(' ');
}

export function docsPages() {
  const categories = parseProviderCategories();
  const labels = providerLabelMap(categories);
  const seen = new Map();

  return docsFiles().map((file) => {
    const slug = normalizeSlug(file);
    if (seen.has(slug)) {
      throw new Error('Duplicate docs slug ' + slug + ' from ' + seen.get(slug) + ' and ' + file);
    }
    seen.set(slug, file);

    const sourcePath = 'docs/' + file;
    const markdown = readUtf8(join(docsDir, file));
    const label = labels.get(slug);
    const heading = firstHeading(markdown);
    const title = label || heading || titleFromSlug(slug);
    const hasSourceH1 = /^#\s+.+$/m.test(markdown);
    const description = label
      ? 'Import ' + label + ' resources into Terraform configuration and state with Terraformer.'
      : shortDescription(markdown, 'Terraformer documentation for ' + title + '.');

    return {
      file,
      sourcePath,
      slug,
      route: 'docs/' + slug,
      title,
      description,
      markdown,
      hasSourceH1,
      isProvider: labels.has(slug),
    };
  });
}

export function getSidebar() {
  const categories = parseProviderCategories();
  const providerSlugs = new Set(categories.flatMap((category) => category.items.map((item) => item.slug)));
  const referencePages = docsPages()
    .filter((page) => !providerSlugs.has(page.slug))
    .map((page) => ({ label: page.title, slug: page.route }));

  return [
    {
      label: 'Getting Started',
      items: [
        { label: 'Overview', slug: 'docs' },
        { label: 'Installation', slug: 'docs/install' },
        { label: 'Providers', slug: 'docs/providers' },
      ],
    },
    ...categories.map((category) => ({
      label: category.label,
      collapsed: category.label !== 'Major Cloud',
      items: category.items.map((item) => ({
        label: item.label,
        slug: 'docs/' + item.slug,
      })),
    })),
    {
      label: 'Reference',
      items: referencePages,
    },
  ];
}

export function knownRoutes() {
  const routes = new Map([
    ['README.md', ''],
    ['readme.md', ''],
    ['docs', 'docs'],
    ['docs/', 'docs'],
    ['/docs', 'docs'],
    ['/docs/', 'docs'],
  ]);

  for (const page of docsPages()) {
    routes.set(page.sourcePath, page.route);
    routes.set('/' + page.sourcePath, page.route);
    routes.set(page.file, page.route);
    routes.set('./' + page.file, page.route);
    routes.set(page.slug, page.route);
  }

  routes.set('docs/index.md', 'docs');
  routes.set('/docs/index.md', 'docs');
  routes.set('docs/install.md', 'docs/install');
  routes.set('/docs/install.md', 'docs/install');
  routes.set('docs/providers.md', 'docs/providers');
  routes.set('/docs/providers.md', 'docs/providers');

  return routes;
}

export function routeToHref(fromRoute, toRoute, anchor = '') {
  if (fromRoute === toRoute) {
    return anchor || './';
  }

  const fromParts = fromRoute ? fromRoute.split('/') : [];
  const toParts = toRoute ? toRoute.split('/') : [];
  let common = 0;

  while (
    common < fromParts.length &&
    common < toParts.length &&
    fromParts[common] === toParts[common]
  ) {
    common += 1;
  }

  const up = '../'.repeat(fromParts.length - common);
  const downParts = toParts.slice(common);
  const down = downParts.length > 0 ? downParts.join('/') + '/' : '';
  const relative = up + down;

  return (relative.startsWith('../') ? relative : './' + relative) + anchor;
}

export function splitUrl(url) {
  const match = /^([^#?]*)([?#].*)?$/.exec(url);
  return {
    path: match?.[1] ?? url,
    suffix: match?.[2] ?? '',
  };
}

export function isExternalUrl(url) {
  return /^(?:[a-z][a-z0-9+.-]*:|\/\/)/i.test(url);
}

function repoPathForLink(path, sourcePath) {
  const sourceDir = sourcePath.includes('/') ? sourcePath.split('/').slice(0, -1).join('/') : '';
  return posix.normalize(posix.join(sourceDir, path)).replace(/^(\.\.\/)+/, '');
}

function githubBlobUrl(repoPath, suffix) {
  return 'https://github.com/chenrui333/terraformer/blob/main/' + repoPath + suffix;
}

export function rewriteMarkdownLinks(markdown, fromRoute, sourcePath = 'README.md') {
  const routes = knownRoutes();

  return markdown.replace(/(!?)\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g, (full, bang, label, rawUrl) => {
    if (bang || isExternalUrl(rawUrl) || rawUrl.startsWith('#')) {
      return full;
    }

    const { path, suffix } = splitUrl(rawUrl);
    const normalizedPath = path.replace(/^\.\//, '');
    const resolvedPath = repoPathForLink(path, sourcePath);
    const targetRoute =
      routes.get(path) ??
      routes.get(normalizedPath) ??
      routes.get(resolvedPath) ??
      routes.get('/' + resolvedPath);

    if (!targetRoute && targetRoute !== '') {
      if (existsSync(join(repoRoot, resolvedPath))) {
        return '[' + label + '](' + githubBlobUrl(resolvedPath, suffix) + ')';
      }
      return full;
    }

    return '[' + label + '](' + routeToHref(fromRoute, targetRoute, suffix) + ')';
  });
}
