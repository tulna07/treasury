import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  transpilePackages: ["@treasury/shared", "@treasury/ui"],
  experimental: {
    optimizePackageImports: ["lucide-react", "@tabler/icons-react"],
  },
  turbopack: {
    resolveAlias: {
      "tw-animate-css": "./node_modules/tw-animate-css/dist/tw-animate.css",
      "shadcn/tailwind.css": "./node_modules/shadcn/dist/tailwind.css",
    },
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "X-DNS-Prefetch-Control", value: "on" },
          {
            key: "Strict-Transport-Security",
            value: "max-age=63072000; includeSubDomains; preload",
          },
          {
            key: "Permissions-Policy",
            value:
              "camera=(), microphone=(), geolocation=(), browsing-topics=()",
          },
        ],
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:34080/api/:path*",
      },
    ];
  },
};

export default nextConfig;
