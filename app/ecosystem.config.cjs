module.exports = {
  apps: [
    {
      name: "treasury-api",
      cwd: "./services/api",
      script: "./bin/treasury-api",
      env: {
        APP_PORT: "34080",
        APP_ENV: "development",
      },
    },
    {
      name: "treasury-web",
      cwd: "./apps/web",
      script: "npx",
      args: "next start --port 34000",
      interpreter: "none",
      env: {
        NODE_ENV: "production",
      },
    },
  ],
};
