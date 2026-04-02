import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  tutorialSidebar: [
    'overview',
    'quick-start',
    'authentication',
    {
      type: 'category',
      label: 'Workflows',
      items: [
        'workflows/login',
        'workflows/create-rule',
        'workflows/test-event-dryrun',
        'workflows/test-event-execute',
        'workflows/update-points',
        'workflows/assign-badge',
        'workflows/read-analytics',
      ],
    },
    {
      type: 'category',
      label: 'Rules API',
      items: [
        'rules/list-rules',
        'rules/create-rule',
        'rules/update-rule',
        'rules/delete-rule',
      ],
    },
    {
      type: 'category',
      label: 'Users API',
      items: [
        'users/list-users',
        'users/get-user-profile',
        'users/update-user-points',
        'users/assign-badge',
      ],
    },
    {
      type: 'category',
      label: 'Badges API',
      items: [
        'badges/list-badges',
        'badges/create-badge',
      ],
    },
    {
      type: 'category',
      label: 'Events API',
      items: [
        'events/test-event',
      ],
    },
    {
      type: 'category',
      label: 'Analytics API',
      items: [
        'analytics/summary',
        'analytics/activity',
      ],
    },
    'error-handling',
  ],
};

export default sidebars;