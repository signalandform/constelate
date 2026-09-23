import type { Metadata } from "next";
import Link from "next/link";
import { Bricolage_Grotesque, IBM_Plex_Mono } from "next/font/google";
import { GitHubMark } from "@/components/GitHubMark";
import "./globals.css";

const bricolage = Bricolage_Grotesque({
  variable: "--font-bricolage",
  subsets: ["latin"],
  axes: ["opsz", "wdth"],
});

const plexMono = IBM_Plex_Mono({
  variable: "--font-plex-mono",
  subsets: ["latin"],
  weight: ["400", "600"],
});

export const metadata: Metadata = {
  metadataBase: new URL("https://constelate.dev"),
  title: {
    default: "Const.Elate — a character sheet for Claude Code",
    template: "%s — Const.Elate",
  },
  description:
    "A terminal character sheet and skill tree for Claude Code. It edits the plain files Claude Code already reads, shows every change first, and never runs an agent of its own.",
  openGraph: {
    title: "Const.Elate",
    description: "A character sheet and skill tree for Claude Code, in the terminal.",
    url: "https://constelate.dev",
    siteName: "Const.Elate",
    type: "website",
  },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${bricolage.variable} ${plexMono.variable} h-full`}>
      <body className="min-h-full flex flex-col">
        <header className="nav">
          <div className="wrap nav-inner">
            <Link href="/" className="wordmark">
              <span className="star" aria-hidden="true">
                ★
              </span>
              Const.Elate
            </Link>
            <nav className="nav-links" aria-label="Site">
              <Link href="/docs">Docs</Link>
              <a href="https://github.com/signalandform/constelate" className="with-mark">
                <GitHubMark />
                GitHub
              </a>
            </nav>
          </div>
        </header>
        <main className="flex-1">{children}</main>
        <footer className="footer">
          <div className="wrap footer-inner">
            <div className="footer-maker">
              <a
                href="https://www.signalandform.net"
                className="sf-logo"
                aria-label="Signal & Form, web and SEO studio in Grapevine, Texas"
              >
                <span className="sf-logo-mask" />
              </a>
              <p>
                Const.Elate is made by{" "}
                <a href="https://www.signalandform.net">Signal &amp; Form</a>, a web and SEO
                studio in Grapevine, Texas.
              </p>
            </div>
            <div className="footer-links">
              <a href="https://github.com/signalandform/constelate" className="with-mark">
                <GitHubMark />
                signalandform/constelate
              </a>
              <p className="small">
                Source, issues and releases live on GitHub. Not affiliated with Anthropic;
                Claude Code is Anthropic&rsquo;s.
              </p>
            </div>
          </div>
        </footer>
      </body>
    </html>
  );
}
