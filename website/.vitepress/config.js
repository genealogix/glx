import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'GENEALOGIX',
  description: 'A modern, evidence-first, Git-native genealogy data standard',

  // Set the base URL for Cloudflare Pages deployment
  base: '/',

  // Set source directory to parent (repository root)
  // This allows VitePress to access all markdown files in the repo
  srcDir: '..',

  // Ignore dead links temporarily during setup
  ignoreDeadLinks: true,

  // Head configuration
  head: [
    ['link', { rel: 'icon', href: '/logo.svg', type: 'image/svg+xml' }]
  ],

  // Vite configuration for file watching in Docker/WSL
  vite: {
    server: {
      watch: {
        usePolling: true,
        interval: 100
      }
    },
    build: {
      rollupOptions: {
        external: ['vue', 'vue/server-renderer']
      }
    }
  },

  // Rewrite paths to map source directories to desired URL structure
  // Paths are now relative to srcDir (parent directory)
  // IMPORTANT: Specific rewrites must come BEFORE wildcards
  rewrites: {
    // Website homepage
    'website/index.md': 'index.md',

    // Docs section - specific files first
    'docs/quickstart.md': 'quickstart.md',
    'docs/examples/README.md': 'examples/index.md',

    // Root-level docs to development section
    'CONTRIBUTING.md': 'development/contributing.md',
    'CODE_OF_CONDUCT.md': 'development/code-of-conduct.md',

    // GLX CLI documentation
    'glx/README.md': 'cli.md',

    'docs/examples/basic-family/README.md': 'examples/basic-family/index.md',
    'docs/examples/complete-family/README.md': 'examples/complete-family/index.md',
    'docs/examples/minimal/README.md': 'examples/minimal/index.md',
    'docs/examples/single-file/README.md': 'examples/single-file/index.md',
    'docs/examples/temporal-properties/README.md': 'examples/temporal-properties/index.md',
    'docs/examples/participant-assertions/README.md': 'examples/participant-assertions/index.md',
    'docs/guides/:page*': 'guides/:page*',
    'docs/development/:page*': 'development/:page*',
    'docs/examples/:page*': 'examples/:page*',

    // Specification section - specific files first, then wildcards
    'specification/README.md': 'specification/index.md',
    'specification/schema/README.md': 'specification/schema/index.md',
    'specification/4-entity-types/README.md': 'specification/4-entity-types/index.md',
    'specification/5-standard-vocabularies/README.md':
      'specification/5-standard-vocabularies/index.md',
    'specification/:page*': 'specification/:page*'
  },

  themeConfig: {
    // Site logo and branding
    logo: '/logo.svg',
    siteTitle: 'GENEALOGIX',

    // Navigation bar
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Quickstart', link: '/quickstart' },
      { text: 'CLI', link: '/cli' },
      {
        text: 'Specification',
        items: [
          { text: 'Overview', link: '/specification/' },
          { text: 'Introduction', link: '/specification/1-introduction' },
          { text: 'Core Concepts', link: '/specification/2-core-concepts' },
          { text: 'Entity Types', link: '/specification/4-entity-types/' },
          { text: 'Standard Vocabularies', link: '/specification/5-standard-vocabularies/' },
          { text: 'JSON Schemas', link: '/specification/schema/' }
        ]
      },
      {
        text: 'Guides',
        items: [
          { text: 'Best Practices', link: '/guides/best-practices' },
          { text: 'Migration from GEDCOM', link: '/guides/migration-from-gedcom' },
          { text: 'Glossary', link: '/guides/glossary' }
        ]
      },
      { text: 'Examples', link: '/examples/' },
      {
        text: 'Development',
        items: [
          { text: 'Architecture', link: '/development/architecture' },
          { text: 'Setup', link: '/development/setup' },
          { text: 'Testing Guide', link: '/development/testing-guide' },
          { text: 'Schema Development', link: '/development/schema-development' },
          { text: 'GEDCOM Import', link: '/development/gedcom-import' },
          { text: 'Contributing Guide', link: '/development/contributing' },
          { text: 'Code of Conduct', link: '/development/code-of-conduct' }
        ]
      },
      {
        text: 'Links',
        items: [
          { text: 'GitHub', link: 'https://github.com/genealogix/glx' },
          { text: 'Discussions', link: 'https://github.com/genealogix/glx/discussions' },
          { text: 'Issues', link: 'https://github.com/genealogix/glx/issues' }
        ]
      }
    ],

    // Sidebar configuration
    sidebar: {
      '/specification/': [
        {
          text: 'Specification',
          items: [
            { text: 'Overview', link: '/specification/' },
            { text: 'Introduction', link: '/specification/1-introduction' },
            { text: 'Core Concepts', link: '/specification/2-core-concepts' },
            { text: 'Archive Organization', link: '/specification/3-archive-organization' }
          ]
        },
        {
          text: 'Entity Types',
          items: [
            { text: 'Overview', link: '/specification/4-entity-types/' },
            { text: 'Vocabularies', link: '/specification/4-entity-types/vocabularies' },
            { text: 'Person', link: '/specification/4-entity-types/person' },
            { text: 'Relationship', link: '/specification/4-entity-types/relationship' },
            { text: 'Event', link: '/specification/4-entity-types/event' },
            { text: 'Place', link: '/specification/4-entity-types/place' },
            { text: 'Source', link: '/specification/4-entity-types/source' },
            { text: 'Citation', link: '/specification/4-entity-types/citation' },
            { text: 'Assertion', link: '/specification/4-entity-types/assertion' },
            { text: 'Repository', link: '/specification/4-entity-types/repository' },
            { text: 'Media', link: '/specification/4-entity-types/media' }
          ]
        },
        {
          text: 'Standard Vocabularies',
          link: '/specification/5-standard-vocabularies/'
        },
        {
          text: 'Schemas',
          items: [{ text: 'JSON Schemas', link: '/specification/schema/' }]
        }
      ],
      '/specification/5-standard-vocabularies/': [
        {
          text: 'Specification',
          items: [
            { text: 'Overview', link: '/specification/' },
            { text: 'Introduction', link: '/specification/1-introduction' },
            { text: 'Core Concepts', link: '/specification/2-core-concepts' },
            { text: 'Archive Organization', link: '/specification/3-archive-organization' }
          ]
        },
        {
          text: 'Entity Types',
          items: [
            { text: 'Overview', link: '/specification/4-entity-types/' },
            { text: 'Vocabularies', link: '/specification/4-entity-types/vocabularies' },
            { text: 'Person', link: '/specification/4-entity-types/person' },
            { text: 'Relationship', link: '/specification/4-entity-types/relationship' },
            { text: 'Event', link: '/specification/4-entity-types/event' },
            { text: 'Place', link: '/specification/4-entity-types/place' },
            { text: 'Source', link: '/specification/4-entity-types/source' },
            { text: 'Citation', link: '/specification/4-entity-types/citation' },
            { text: 'Assertion', link: '/specification/4-entity-types/assertion' },
            { text: 'Repository', link: '/specification/4-entity-types/repository' },
            { text: 'Media', link: '/specification/4-entity-types/media' }
          ]
        },
        {
          text: 'Standard Vocabularies',
          link: '/specification/5-standard-vocabularies/'
        },
        {
          text: 'Schemas',
          items: [{ text: 'JSON Schemas', link: '/specification/schema/' }]
        }
      ],
      '/guides/': [
        {
          text: 'User Guides',
          items: [
            { text: 'Best Practices', link: '/guides/best-practices' },
            { text: 'Migration from GEDCOM', link: '/guides/migration-from-gedcom' },
            { text: 'Glossary', link: '/guides/glossary' }
          ]
        }
      ],
      '/development/': [
        {
          text: 'Developer Guides',
          items: [
            { text: 'Architecture', link: '/development/architecture' },
            { text: 'Setup', link: '/development/setup' },
            { text: 'Testing Guide', link: '/development/testing-guide' },
            { text: 'Schema Development', link: '/development/schema-development' },
            { text: 'GEDCOM Import', link: '/development/gedcom-import' }
          ]
        },
        {
          text: 'Contributing',
          items: [
            { text: 'Contributing Guide', link: '/development/contributing' },
            { text: 'Code of Conduct', link: '/development/code-of-conduct' }
          ]
        }
      ],
      '/examples/': [
        {
          text: 'Examples',
          items: [{ text: 'Overview', link: '/examples/' }]
        },
        {
          text: 'For Beginners',
          items: [
            { text: 'Minimal', link: '/examples/minimal/' },
            { text: 'Basic Family', link: '/examples/basic-family/' },
            { text: 'Complete Family ⭐', link: '/examples/complete-family/' }
          ]
        },
        {
          text: 'Advanced Concepts',
          items: [
            { text: 'Single-File Archives', link: '/examples/single-file/' },
            { text: 'Temporal Properties', link: '/examples/temporal-properties/' },
            { text: 'Participant Assertions', link: '/examples/participant-assertions/' }
          ]
        }
      ]
    },

    // Social links
    socialLinks: [{ icon: 'github', link: 'https://github.com/genealogix/glx' }],

    // Footer
    footer: {
      message: 'Licensed under Apache License 2.0',
      copyright: 'Copyright © 2025 Oracynth, Inc.'
    },

    // Edit link
    editLink: {
      pattern: 'https://github.com/genealogix/glx/edit/main/:path',
      text: 'Edit this page on GitHub'
    },

    // Last updated timestamp
    lastUpdated: {
      text: 'Last updated',
      formatOptions: {
        dateStyle: 'medium',
        timeStyle: 'short'
      }
    },

    // Search
    search: {
      provider: 'local'
    }
  }
})
