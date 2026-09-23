import { DocsNav } from "@/components/DocsNav";

export default function DocsLayout({ children }: LayoutProps<"/docs">) {
  return (
    <div className="wrap docs">
      <DocsNav />
      <article className="prose">{children}</article>
    </div>
  );
}
