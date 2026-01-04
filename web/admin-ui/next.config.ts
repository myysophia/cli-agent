import type { NextConfig } from "next";

const basePath = "/v1/admin";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "export",
  trailingSlash: true,
  basePath,
  assetPrefix: `${basePath}/`,
  images: {
    unoptimized: true
  }
};

export default nextConfig;
