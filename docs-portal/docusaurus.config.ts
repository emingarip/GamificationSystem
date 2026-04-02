import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Gamification API Documentation',
  tagline: 'AI-Native Gamification Platform with Knowledge Graph',
  favicon: 'img/logo.svg',

  url: 'https://gamification.example.com',
  baseUrl: '/docs/',

  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/example/gamification-system/tree/main/docs-portal/',
          showLastUpdateTime: false,
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  // Note: redocusaurus plugin removed due to compatibility issues with Docusaurus v3
  // API reference is available via Swagger UI at /swagger/index.html

  themeConfig: {
    navbar: {
      title: 'Gamification API',
      logo: {
        alt: 'Gamification Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          to: '/',
          label: 'Docs',
          position: 'left',
        },
        {
          to: '/api-reference',
          label: 'API Ref',
          position: 'left',
        },
        {
          type: 'search',
          position: 'right',
        },
        {
          href: 'https://github.com/example/gamification-system',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      copyright: `Copyright © ${new Date().getFullYear()} Gamification System. Built with Docusaurus.`,
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Overview',
              to: '/',
            },
            {
              label: 'API Reference',
              to: '/api-reference',
            },
          ],
        },
      ],
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
    // Algolia search disabled - add credentials to enable
    // algolia: {
    //   appId: 'YOUR_APP_ID',
    //   apiKey: 'YOUR_SEARCH_API_KEY',
    //   indexName: 'gamification',
    // },
  } satisfies Preset.ThemeConfig,
};

export default config;
