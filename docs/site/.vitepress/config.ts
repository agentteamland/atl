import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: 'AgentTeamLand',
  description: 'AI agent teams, installed like packages.',

  // Served at docs.agentteamland.com via a GitHub Pages custom domain (root path).
  base: '/',

  lastUpdated: true,
  cleanUrls: true,
  metaChunk: true,

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['meta', { name: 'theme-color', content: '#3b6df7' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'AgentTeamLand' }],
    ['meta', { property: 'og:description', content: 'AI agent teams, installed like packages.' }],
    ['meta', { property: 'og:image', content: 'https://raw.githubusercontent.com/agentteamland/workspace/main/assets/demo.gif' }]
  ],

  // ---------------------------------------------------------------------------
  // i18n — English is canonical at /, Turkish at /tr/
  // ---------------------------------------------------------------------------
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/what-is-atl', activeMatch: '/guide/' },
          { text: 'CLI', link: '/cli/overview', activeMatch: '/cli/' },
          { text: 'Skills', link: '/skills/drain', activeMatch: '/skills/' },
          { text: 'Teams', link: '/teams/', activeMatch: '/teams/' },
          { text: 'Team Authoring', link: '/authoring/team-json', activeMatch: '/authoring/' },
          { text: 'Contributing', link: '/contributing/workspace', activeMatch: '/contributing/' },
          { text: 'Reference', link: '/reference/faq', activeMatch: '/reference/' },
          {
            text: 'Ecosystem',
            items: [
              { text: 'GitHub org', link: 'https://github.com/agentteamland' },
              { text: 'atl monorepo', link: 'https://github.com/agentteamland/atl' },
              { text: 'Workspace', link: 'https://github.com/agentteamland/workspace' }
            ]
          }
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Guide',
              items: [
                { text: 'What is atl?', link: '/guide/what-is-atl' },
                { text: 'Install', link: '/guide/install' },
                { text: 'Quickstart', link: '/guide/quickstart' },
                { text: 'Concepts', link: '/guide/concepts' },
                { text: 'Karpathy guidelines', link: '/guide/karpathy-guidelines' },
                { text: 'Skill selection discipline', link: '/guide/skill-selection-discipline' }
              ]
            },
            {
              text: 'Knowledge model',
              items: [
                { text: 'Knowledge system', link: '/guide/knowledge-system' },
                { text: 'Children + learnings', link: '/guide/children-and-learnings' },
                { text: 'Learning marker lifecycle', link: '/guide/learning-marker-lifecycle' },
                { text: 'Claude Code conventions', link: '/guide/claude-code-conventions' }
              ]
            },
            {
              text: 'Operations',
              items: [
                { text: 'Governance', link: '/guide/governance' }
              ]
            }
          ],
          '/cli/': [
            {
              text: 'CLI Reference',
              items: [
                { text: 'Overview', link: '/cli/overview' },
                { text: 'atl install', link: '/cli/install' },
                { text: 'atl search', link: '/cli/search' },
                { text: 'atl list', link: '/cli/list' },
                { text: 'atl remove', link: '/cli/remove' },
                { text: 'atl update', link: '/cli/update' },
                { text: 'atl promote', link: '/cli/promote' },
                { text: 'atl pin / unpin', link: '/cli/pin' },
                { text: 'atl tick', link: '/cli/tick' },
                { text: 'atl learnings', link: '/cli/learnings' },
                { text: 'atl doctor', link: '/cli/doctor' },
                { text: 'atl publish', link: '/cli/publish' },
                { text: 'atl setup-hooks', link: '/cli/setup-hooks' }
              ]
            }
          ],
          '/skills/': [
            {
              text: 'Global Skills',
              items: [
                { text: '/drain', link: '/skills/drain' },
                { text: '/wiki', link: '/skills/wiki' },
                { text: '/brainstorm', link: '/skills/brainstorm' },
                { text: '/rule', link: '/skills/rule' },
                { text: '/rule-wizard', link: '/skills/rule-wizard' },
                { text: '/create-pr', link: '/skills/create-pr' },
                { text: '/create-code-diagram', link: '/skills/create-code-diagram' }
              ]
            }
          ],
          '/teams/': [
            {
              text: 'Verified Teams',
              items: [
                { text: 'Browse', link: '/teams/' },
                { text: 'software-project-team', link: '/teams/software-project-team' },
                { text: 'design-system-team', link: '/teams/design-system-team' }
              ]
            }
          ],
          '/authoring/': [
            {
              text: 'Team Authoring',
              items: [
                { text: 'team.json', link: '/authoring/team-json' },
                { text: 'Creating a team', link: '/authoring/creating-a-team' },
                { text: 'Scaffolder spec', link: '/authoring/scaffolder-spec' }
              ]
            }
          ],
          '/contributing/': [
            {
              text: 'Contributing',
              items: [
                { text: 'Workspace (maintainer hub)', link: '/contributing/workspace' },
                { text: 'Release pipeline', link: '/contributing/release-pipeline' }
              ]
            }
          ],
          '/reference/': [
            {
              text: 'Reference',
              items: [
                { text: 'Schema', link: '/reference/schema' },
                { text: 'Glossary', link: '/reference/glossary' },
                { text: 'FAQ', link: '/reference/faq' }
              ]
            }
          ]
        }
      }
    },
    tr: {
      label: 'Türkçe',
      lang: 'tr',
      link: '/tr/',
      themeConfig: {
        nav: [
          { text: 'Rehber', link: '/tr/guide/what-is-atl', activeMatch: '/tr/guide/' },
          { text: 'CLI', link: '/tr/cli/overview', activeMatch: '/tr/cli/' },
          { text: 'Skill\'ler', link: '/tr/skills/drain', activeMatch: '/tr/skills/' },
          { text: 'Takımlar', link: '/tr/teams/', activeMatch: '/tr/teams/' },
          { text: 'Takım Yazımı', link: '/tr/authoring/team-json', activeMatch: '/tr/authoring/' },
          { text: 'Katkı', link: '/tr/contributing/workspace', activeMatch: '/tr/contributing/' },
          { text: 'Başvuru', link: '/tr/reference/faq', activeMatch: '/tr/reference/' },
          {
            text: 'Ekosistem',
            items: [
              { text: 'GitHub org', link: 'https://github.com/agentteamland' },
              { text: 'atl monorepo', link: 'https://github.com/agentteamland/atl' },
              { text: 'Workspace', link: 'https://github.com/agentteamland/workspace' }
            ]
          }
        ],
        sidebar: {
          '/tr/guide/': [
            {
              text: 'Rehber',
              items: [
                { text: 'atl nedir?', link: '/tr/guide/what-is-atl' },
                { text: 'Kurulum', link: '/tr/guide/install' },
                { text: 'Hızlı başlangıç', link: '/tr/guide/quickstart' },
                { text: 'Kavramlar', link: '/tr/guide/concepts' },
                { text: 'Karpathy ilkeleri', link: '/tr/guide/karpathy-guidelines' },
                { text: 'Skill seçim disiplini', link: '/tr/guide/skill-selection-discipline' }
              ]
            },
            {
              text: 'Bilgi modeli',
              items: [
                { text: 'Bilgi sistemi', link: '/tr/guide/knowledge-system' },
                { text: 'Children + learnings', link: '/tr/guide/children-and-learnings' },
                { text: 'Öğrenme işaretçisi yaşam döngüsü', link: '/tr/guide/learning-marker-lifecycle' },
                { text: 'Claude Code sözleşmeleri', link: '/tr/guide/claude-code-conventions' }
              ]
            },
            {
              text: 'İşletim',
              items: [
                { text: 'Yönetişim', link: '/tr/guide/governance' }
              ]
            }
          ],
          '/tr/cli/': [
            {
              text: 'CLI Başvuru',
              items: [
                { text: 'Genel bakış', link: '/tr/cli/overview' },
                { text: 'atl install', link: '/tr/cli/install' },
                { text: 'atl search', link: '/tr/cli/search' },
                { text: 'atl list', link: '/tr/cli/list' },
                { text: 'atl remove', link: '/tr/cli/remove' },
                { text: 'atl update', link: '/tr/cli/update' },
                { text: 'atl promote', link: '/tr/cli/promote' },
                { text: 'atl pin / unpin', link: '/tr/cli/pin' },
                { text: 'atl tick', link: '/tr/cli/tick' },
                { text: 'atl learnings', link: '/tr/cli/learnings' },
                { text: 'atl doctor', link: '/tr/cli/doctor' },
                { text: 'atl publish', link: '/tr/cli/publish' },
                { text: 'atl setup-hooks', link: '/tr/cli/setup-hooks' }
              ]
            }
          ],
          '/tr/skills/': [
            {
              text: 'Global Skill\'ler',
              items: [
                { text: '/drain', link: '/tr/skills/drain' },
                { text: '/wiki', link: '/tr/skills/wiki' },
                { text: '/brainstorm', link: '/tr/skills/brainstorm' },
                { text: '/rule', link: '/tr/skills/rule' },
                { text: '/rule-wizard', link: '/tr/skills/rule-wizard' },
                { text: '/create-pr', link: '/tr/skills/create-pr' },
                { text: '/create-code-diagram', link: '/tr/skills/create-code-diagram' }
              ]
            }
          ],
          '/tr/teams/': [
            {
              text: 'Onaylı Takımlar',
              items: [
                { text: 'Göz at', link: '/tr/teams/' },
                { text: 'software-project-team', link: '/tr/teams/software-project-team' },
                { text: 'design-system-team', link: '/tr/teams/design-system-team' }
              ]
            }
          ],
          '/tr/authoring/': [
            {
              text: 'Takım Yazımı',
              items: [
                { text: 'team.json', link: '/tr/authoring/team-json' },
                { text: 'Takım oluşturma', link: '/tr/authoring/creating-a-team' },
                { text: 'İskele belirtimi', link: '/tr/authoring/scaffolder-spec' }
              ]
            }
          ],
          '/tr/contributing/': [
            {
              text: 'Katkı',
              items: [
                { text: 'Workspace (maintainer hub)', link: '/tr/contributing/workspace' },
                { text: 'Release pipeline', link: '/tr/contributing/release-pipeline' }
              ]
            }
          ],
          '/tr/reference/': [
            {
              text: 'Başvuru',
              items: [
                { text: 'Şema', link: '/tr/reference/schema' },
                { text: 'Sözlük', link: '/tr/reference/glossary' },
                { text: 'SSS', link: '/tr/reference/faq' }
              ]
            }
          ]
        },
        outline: { label: 'Bu sayfada' },
        docFooter: { prev: 'Önceki', next: 'Sonraki' },
        lastUpdatedText: 'Son güncelleme',
        darkModeSwitchLabel: 'Tema',
        lightModeSwitchTitle: 'Açık temaya geç',
        darkModeSwitchTitle: 'Koyu temaya geç',
        sidebarMenuLabel: 'Menü',
        returnToTopLabel: 'Başa dön',
        externalLinkIcon: true
      }
    }
  },

  themeConfig: {
    logo: { src: '/logo.svg', width: 24, height: 24, alt: 'AgentTeamLand' },
    siteTitle: 'AgentTeamLand',

    socialLinks: [
      { icon: 'github', link: 'https://github.com/agentteamland' }
    ],

    search: {
      provider: 'local',
      options: {
        locales: {
          tr: {
            translations: {
              button: { buttonText: 'Ara', buttonAriaLabel: 'Ara' },
              modal: {
                displayDetails: 'Ayrıntıları göster',
                resetButtonTitle: 'Sorguyu temizle',
                backButtonTitle: 'Aramayı kapat',
                noResultsText: 'Sonuç bulunamadı',
                footer: {
                  selectText: 'seç',
                  navigateText: 'gez',
                  closeText: 'kapat'
                }
              }
            }
          }
        }
      }
    },

    editLink: {
      pattern: 'https://github.com/agentteamland/atl/edit/main/docs/site/:path',
      text: 'Edit this page on GitHub'
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 AgentTeamLand'
    }
  }
})
