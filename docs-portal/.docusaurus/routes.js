import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/docs/api-reference',
    component: ComponentCreator('/docs/api-reference', '44b'),
    exact: true
  },
  {
    path: '/docs/',
    component: ComponentCreator('/docs/', '4a4'),
    routes: [
      {
        path: '/docs/',
        component: ComponentCreator('/docs/', '4df'),
        routes: [
          {
            path: '/docs/',
            component: ComponentCreator('/docs/', '93e'),
            routes: [
              {
                path: '/docs/analytics/activity',
                component: ComponentCreator('/docs/analytics/activity', '770'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/analytics/summary',
                component: ComponentCreator('/docs/analytics/summary', '99a'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/authentication',
                component: ComponentCreator('/docs/authentication', '938'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/badges/create-badge',
                component: ComponentCreator('/docs/badges/create-badge', '92c'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/badges/list-badges',
                component: ComponentCreator('/docs/badges/list-badges', 'c97'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/error-handling',
                component: ComponentCreator('/docs/error-handling', 'c9b'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/events/test-event',
                component: ComponentCreator('/docs/events/test-event', '935'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/quick-start',
                component: ComponentCreator('/docs/quick-start', 'b74'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/rules/create-rule',
                component: ComponentCreator('/docs/rules/create-rule', 'f5a'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/rules/delete-rule',
                component: ComponentCreator('/docs/rules/delete-rule', '223'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/rules/list-rules',
                component: ComponentCreator('/docs/rules/list-rules', '21e'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/rules/update-rule',
                component: ComponentCreator('/docs/rules/update-rule', '734'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/users/assign-badge',
                component: ComponentCreator('/docs/users/assign-badge', 'ac3'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/users/get-user-profile',
                component: ComponentCreator('/docs/users/get-user-profile', '084'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/users/list-users',
                component: ComponentCreator('/docs/users/list-users', '911'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/users/update-user-points',
                component: ComponentCreator('/docs/users/update-user-points', 'e51'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/',
                component: ComponentCreator('/docs/workflows/', '6c1'),
                exact: true
              },
              {
                path: '/docs/workflows/assign-badge',
                component: ComponentCreator('/docs/workflows/assign-badge', '28d'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/create-rule',
                component: ComponentCreator('/docs/workflows/create-rule', 'a82'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/login',
                component: ComponentCreator('/docs/workflows/login', 'b12'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/read-analytics',
                component: ComponentCreator('/docs/workflows/read-analytics', 'a77'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/test-event-dryrun',
                component: ComponentCreator('/docs/workflows/test-event-dryrun', 'e1d'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/test-event-execute',
                component: ComponentCreator('/docs/workflows/test-event-execute', 'bfd'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/workflows/update-points',
                component: ComponentCreator('/docs/workflows/update-points', '7d2'),
                exact: true,
                sidebar: "tutorialSidebar"
              },
              {
                path: '/docs/',
                component: ComponentCreator('/docs/', '540'),
                exact: true,
                sidebar: "tutorialSidebar"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
