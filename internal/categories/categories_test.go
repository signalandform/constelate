package categories

import "testing"

func TestSuggest(t *testing.T) {
	cases := map[string]string{
		"Audit and improve web accessibility following WCAG 2.2":       "Design",
		"Master Next.js 14+ App Router with Server Components":         "Coding",
		"Write, rewrite, or improve marketing copy for any page":       "Writing",
		"Optimize for search engine visibility and ranking, SEO audit": "Research",
		"Deploy to Vercel and manage Supabase migrations":              "Ops",
		"": "",
	}
	for desc, want := range cases {
		if got := Suggest("", desc); got != want {
			t.Errorf("Suggest(%q) = %q, want %q", desc, got, want)
		}
	}
}

func TestResolveOrder(t *testing.T) {
	s := Store{File: File{Assign: map[string]string{"personal/seo": "Writing", "copywriting": "Ops"}}}
	if c, sug := s.Resolve("personal", "seo", "search engine"); c != "Writing" || sug {
		t.Errorf("scoped assignment lost: %s %v", c, sug)
	}
	if c, sug := s.Resolve("project", "copywriting", "marketing copy"); c != "Ops" || sug {
		t.Errorf("name assignment lost: %s %v", c, sug)
	}
	if c, sug := s.Resolve("personal", "x", "zzz"); c != Unsorted || !sug {
		t.Errorf("fallback: %s %v", c, sug)
	}
}

func TestSetAddsCustom(t *testing.T) {
	s := Store{File: File{Assign: map[string]string{}}}
	s.Set("personal", "foo", "Games")
	if len(s.File.Custom) != 1 || s.File.Custom[0] != "Games" {
		t.Errorf("custom = %v", s.File.Custom)
	}
	all := s.All()
	if all[len(all)-1] != Unsorted || !contains(all, "Games") {
		t.Errorf("All = %v", all)
	}
}
