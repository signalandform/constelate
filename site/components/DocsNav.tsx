"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const pages = [
  { href: "/docs", title: "Overview" },
  { href: "/docs/install", title: "Install" },
  { href: "/docs/screens", title: "Screens and keys" },
  { href: "/docs/mechanics", title: "Mechanics" },
  { href: "/docs/configuration", title: "Configuration" },
  { href: "/docs/safety", title: "Safety" },
  { href: "/docs/command-line", title: "Command line" },
];

export function DocsNav() {
  const path = usePathname();
  return (
    <nav className="docs-nav" aria-label="Documentation">
      <ol>
        {pages.map((p) => (
          <li key={p.href}>
            <Link href={p.href} aria-current={path === p.href ? "page" : undefined}>
              {p.title}
            </Link>
          </li>
        ))}
      </ol>
    </nav>
  );
}
