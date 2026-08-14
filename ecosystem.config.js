module.exports = {
  apps: [
    {
      name: 'nivra-backend',
      script: './bin/nivra-api',
      cwd: '/var/www/nivra/backend',
      interpreter: 'none',
      autorestart: true,
      max_memory_restart: '300M',
    },
    {
      name: 'nivra-frontend',
      script: 'npm',
      args: 'start',
      cwd: '/var/www/nivra/frontend',
      env: {
        PORT: '3020',
        NODE_ENV: 'production',
      },
      autorestart: true,
      max_memory_restart: '400M',
    },
  ],
};
