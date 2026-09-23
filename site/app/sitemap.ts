import type { MetadataRoute } from "next";

const base = "https://constelate.dev";
const paths = ["", "/docs", "/docs/install", "/docs/screens", "/docs/mechanics", "/docs/configuration", "/docs/safety", "/docs/command-line"];

export default function sitemap(): MetadataRoute.Sitemap {
  return paths.map((p) => ({ url: `${base}${p}`, lastModified: new Date() }));
}
