---
knowledge-base-summary: "Two paths: CSR (Vite+React) for admin panels — minimal SEO, document.title only. SSR (Next.js) for public-facing — full SEO with generateMetadata, OpenGraph, sitemap, structured data. Covers both approaches."
---
# SEO & Meta Tags: CSR vs SSR

## Philosophy

Two fundamentally different paths exist for SEO. The choice depends on who sees the app:

1. **CSR (Vite + React)** -- Admin panels, internal dashboards. No crawlers, no indexing. Minimal SEO: just `document.title` for browser tab UX.
2. **SSR (Next.js)** -- Public-facing apps, landing pages, e-commerce. Full SEO: meta tags, OpenGraph, structured data, sitemap.

Never add SSR complexity to an admin panel. Never ship a public-facing marketing site as a CSR SPA.

## Package version pins (security-critical)

Both CSR and SSR paths pull from npm packages with active CVE histories. Always start a new project from the canonical minimums in [`software-project-team/dependency-versions.md`](../../../dependency-versions.md).

**SSR path — Next.js (≥ 15.5.15):**

```json
"dependencies": {
  "next":      "15.5.15",
  "react":     "19.0.0",
  "react-dom": "19.0.0"
}
```

> **Security note — minimum 15.5.15.** Versions before 15.5.15 carry **2 CRITICAL CVEs** — (1) RCE in React flight protocol, (2) Authorization Bypass in Middleware — plus 4 High and 8 Medium. All fixed in 15.5.15. Discovered while scaffolding a downstream Next.js project; the bug is not project-specific. Any new SSR project starts at 15.5.15 or later; floating range `^15.5.15` lets dependabot deliver future patches.

**CSR path — Vite (≥ 7.x latest patch) + esbuild (≥ 0.25.x latest):**

```json
"devDependencies": {
  "vite":    "^7.0.0",
  "esbuild": "^0.25.0"
}
```

> **Security note.** Vite < 7.x latest had a Path Traversal vulnerability in optimized-deps `.map` handling (medium). esbuild had a dev-server origin-verification CVE (medium). Both fixed; pin to the latest minor in the listed range. See `software-project-team/dependency-versions.md` for the canonical numbers.

`next-intl` (used in i18n integrations on the SSR path) ships `^4.0.0` minimum — fixes an open redirect (medium). See `react-agent/children/i18n.md`.

## Decision Guide

| Criteria | CSR (Vite + React) | SSR (Next.js) |
|----------|-------------------|---------------|
| Audience | Internal users, admin | Public, search engines |
| Crawling needed? | No | Yes |
| Social sharing? | No | Yes (OpenGraph) |
| First paint performance | Acceptable (behind login) | Critical (SEO ranking) |
| Complexity | Low | Higher |
| Use cases | Admin panel, dashboard, CRM | Marketing site, e-commerce, blog |

**Rule of thumb:** If the page requires login to see, use CSR. If Google/social media bots need to see it, use SSR.

---

## Path 1: CSR (Vite + React) -- Admin Panel

### What We Do

Only `document.title` -- so the browser tab shows the current page name. No meta tags, no sitemap, no OpenGraph. Crawlers never see this app (it is behind authentication).

### usePageTitle Hook

```typescript
// src/hooks/use-page-title.ts
import { useEffect } from 'react';

const APP_NAME = 'MyApp Admin';

/**
 * Set the browser tab title for the current page.
 *
 * Usage:
 *   usePageTitle('Orders');        // → "Orders | MyApp Admin"
 *   usePageTitle('Order #1234');   // → "Order #1234 | MyApp Admin"
 */
export function usePageTitle(title: string) {
  useEffect(() => {
    const previousTitle = document.title;
    document.title = `${title} | ${APP_NAME}`;

    return () => {
      document.title = previousTitle;
    };
  }, [title]);
}
```

### Usage in Pages

```typescript
// src/features/orders/pages/orders-page.tsx
import { usePageTitle } from '@/hooks/use-page-title';

export function Component() {
  usePageTitle('Orders');

  return (
    <div>
      <h1>Orders</h1>
      {/* ... */}
    </div>
  );
}

// Dynamic title based on data
export function OrderDetailPage() {
  const { orderId } = useParams();
  const { data: order } = useOrder(orderId);

  usePageTitle(order ? `Order #${order.number}` : 'Loading...');

  return (/* ... */);
}
```

### robots.txt for Admin Panel

Even though the admin is behind auth, add a restrictive robots.txt as a safety measure:

```
# public/robots.txt
User-agent: *
Disallow: /
```

### That Is It for CSR

No `<meta>` tags, no OpenGraph, no sitemap, no structured data. The admin panel is not indexed. Keep it simple.

---

## Path 2: SSR (Next.js) -- Public-Facing App

### Project Setup

```bash
npx create-next-app@latest --typescript --tailwind --app --src-dir
```

### generateMetadata (Per-Page Meta Tags)

Next.js App Router provides `generateMetadata()` for server-side meta tag generation. This runs on the server, so crawlers see the full HTML with all meta tags.

```typescript
// src/app/products/[slug]/page.tsx
import type { Metadata } from 'next';
import { notFound } from 'next/navigation';

interface Props {
  params: Promise<{ slug: string }>;
}

// Server-side metadata generation
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const product = await getProduct(slug);

  if (!product) return {};

  return {
    title: product.name,
    description: product.description.slice(0, 160),
    openGraph: {
      title: product.name,
      description: product.description.slice(0, 160),
      images: [
        {
          url: product.imageUrl,
          width: 1200,
          height: 630,
          alt: product.name,
        },
      ],
      type: 'website',
    },
    twitter: {
      card: 'summary_large_image',
      title: product.name,
      description: product.description.slice(0, 160),
      images: [product.imageUrl],
    },
    alternates: {
      canonical: `https://example.com/products/${slug}`,
    },
  };
}

// Page component
export default async function ProductPage({ params }: Props) {
  const { slug } = await params;
  const product = await getProduct(slug);

  if (!product) notFound();

  return (
    <main>
      <h1>{product.name}</h1>
      {/* ... */}
    </main>
  );
}
```

### Layout-Level Default Metadata

```typescript
// src/app/layout.tsx
import type { Metadata } from 'next';

export const metadata: Metadata = {
  metadataBase: new URL('https://example.com'),
  title: {
    default: 'MyApp - Your Tagline Here',
    template: '%s | MyApp',       // pages override with just a string
  },
  description: 'MyApp helps you do amazing things.',
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: 'https://example.com',
    siteName: 'MyApp',
    images: [
      {
        url: '/og-default.png',
        width: 1200,
        height: 630,
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    site: '@myapp',
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
```

### Structured Data (JSON-LD)

Search engines use structured data for rich snippets (product ratings, prices, breadcrumbs, etc.).

```typescript
// src/components/seo/json-ld.tsx

interface JsonLdProps {
  data: Record<string, unknown>;
}

/**
 * Renders a JSON-LD script tag for structured data.
 * Place inside a page component.
 */
export function JsonLd({ data }: JsonLdProps) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}

// Usage in a product page:
<JsonLd
  data={{
    '@context': 'https://schema.org',
    '@type': 'Product',
    name: product.name,
    description: product.description,
    image: product.imageUrl,
    offers: {
      '@type': 'Offer',
      price: product.price,
      priceCurrency: 'USD',
      availability: product.inStock
        ? 'https://schema.org/InStock'
        : 'https://schema.org/OutOfStock',
    },
  }}
/>

// Usage for an article:
<JsonLd
  data={{
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    author: {
      '@type': 'Person',
      name: article.author.name,
    },
    datePublished: article.publishedAt,
    dateModified: article.updatedAt,
    image: article.coverImage,
  }}
/>

// Breadcrumb structured data:
<JsonLd
  data={{
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://example.com' },
      { '@type': 'ListItem', position: 2, name: 'Products', item: 'https://example.com/products' },
      { '@type': 'ListItem', position: 3, name: product.name },
    ],
  }}
/>
```

### Sitemap Generation

Using `next-sitemap` for automatic sitemap generation after build.

```bash
npm install next-sitemap
```

```javascript
// next-sitemap.config.js
/** @type {import('next-sitemap').IConfig} */
module.exports = {
  siteUrl: 'https://example.com',
  generateRobotsTxt: true,
  sitemapSize: 7000,
  changefreq: 'daily',
  priority: 0.7,
  exclude: ['/admin/*', '/api/*', '/auth/*'],
  robotsTxtOptions: {
    policies: [
      {
        userAgent: '*',
        allow: '/',
        disallow: ['/admin', '/api', '/auth'],
      },
    ],
    additionalSitemaps: [
      'https://example.com/server-sitemap.xml', // for dynamic pages
    ],
  },
};
```

Add to `package.json` scripts:

```json
{
  "scripts": {
    "postbuild": "next-sitemap"
  }
}
```

### Dynamic Sitemap for Database Content

For pages generated from database content (products, articles), create a server-side sitemap:

```typescript
// src/app/server-sitemap.xml/route.ts
import { getServerSideSitemap, type ISitemapField } from 'next-sitemap';

export async function GET() {
  const products = await getAllProductSlugs();

  const fields: ISitemapField[] = products.map((product) => ({
    loc: `https://example.com/products/${product.slug}`,
    lastmod: product.updatedAt,
    changefreq: 'weekly',
    priority: 0.8,
  }));

  return getServerSideSitemap(fields);
}
```

### robots.txt (Generated by next-sitemap)

For manual control:

```typescript
// src/app/robots.ts
import type { MetadataRoute } from 'next';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
        disallow: ['/admin/', '/api/', '/auth/'],
      },
    ],
    sitemap: 'https://example.com/sitemap.xml',
  };
}
```

### Canonical URLs

Prevents duplicate content issues. Every page should have a canonical URL.

```typescript
// In generateMetadata:
alternates: {
  canonical: `https://example.com/products/${slug}`,
  languages: {
    'en': `https://example.com/en/products/${slug}`,
    'tr': `https://example.com/tr/products/${slug}`,
  },
},
```

## Rules

1. **CSR apps (admin panels) get only `document.title`.** No meta tags, no OpenGraph, no sitemap. Don't add complexity where crawlers never visit.
2. **SSR apps get full SEO.** generateMetadata, OpenGraph, JSON-LD, sitemap, robots.txt, canonical URLs.
3. **Description max 160 characters.** Search engines truncate longer descriptions.
4. **OpenGraph image: 1200x630px.** This is the standard for social media previews (Facebook, LinkedIn, Twitter).
5. **Canonical URLs are mandatory.** Every public page must have a canonical URL to prevent duplicate content penalties.
6. **Structured data for rich snippets.** Products, articles, breadcrumbs -- use JSON-LD format, not microdata.
7. **Dynamic sitemap for database content.** Static sitemap for known pages, server-side sitemap for database-driven pages (products, articles).
