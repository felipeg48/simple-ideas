package render

import (
	"strings"
	"testing"
)

func TestHighlightLanguages(t *testing.T) {
	cases := []struct{ lang, code, want string }{
		{"csharp", `public sealed class Order { }`, `<span class="si-tk-k">sealed</span>`},
		{"cs", `namespace Shop;`, `<span class="si-tk-k">namespace</span>`},
		{"c#", `var x = await Run(); // done`, `<span class="si-tk-c">// done</span>`},
		{"xml", `<Project Sdk="Microsoft.NET.Sdk">`, `<span class="si-tk-k">&lt;Project</span>`},
		{"csproj", `<PackageReference Include="RabbitMQ.Client" />`, `<span class="si-tk-y">Include=</span>`},
		{"xml", `<?xml version="1.0"?>`, `<span class="si-tk-m">&lt;?xml version=&#34;1.0&#34;?&gt;</span>`},
		{"xml", `<!-- note -->`, `<span class="si-tk-c">&lt;!-- note --&gt;</span>`},
		{"dockerfile", `FROM alpine AS build`, `<span class="si-tk-k">FROM</span>`},
		{"cpp", `std::vector<int> v; // ok`, `<span class="si-tk-k">int</span>`},
		{"ruby", `def go # hi`, `<span class="si-tk-c"># hi</span>`},
		{"golang", `func main() {}`, `<span class="si-tk-k">func</span>`},
		{"tf", `resource "aws_s3_bucket" "logs" { # the bucket`, `<span class="si-tk-k">resource</span>`},
		{"hcl", `  bucket = var.name`, `<span class="si-tk-y">  bucket =</span>`},
		{"hcl", `// note`, `<span class="si-tk-c">// note</span>`},
		{"proto", `message Order { repeated string items = 1; }`, `<span class="si-tk-k">message</span>`},
		{"proto", `message Order {}`, `<span class="si-tk-t">Order</span>`},
		{"gql", `query Orders($first: Int) { orders }`, `<span class="si-tk-v">$first</span>`},
		{"graphql", `# a comment`, `<span class="si-tk-c"># a comment</span>`},
		{"fs", `let total = List.sum xs (* sum *)`, `<span class="si-tk-c">(* sum *)</span>`},
		{"fsharp", `let mutable x = 1`, `<span class="si-tk-k">mutable</span>`},
		{"exs", `defmodule Shop do`, `<span class="si-tk-k">defmodule</span>`},
		{"ex", `{:ok, order}`, `<span class="si-tk-v">:ok</span>`},
	}
	csharp := "builder.Services.AddCockroachDB(config, pipeline =>\n{\n    pipeline.UseRetry();        // retry on 40001 with exponential backoff\n    // pipeline.UseCircuitBreaker();\n});"
	for _, want := range []string{`<span class="si-tk-f">AddCockroachDB</span>(`, `<span class="si-tk-f">UseRetry</span>()`, `<span class="si-tk-t">Services</span>`,
		`<span class="si-tk-c">// retry on 40001 with exponential backoff</span>`, `<span class="si-tk-c">// pipeline.UseCircuitBreaker();</span>`} {
		if got := highlight(csharp, "csharp"); !strings.Contains(got, want) {
			t.Errorf("csharp: missing %s in\n%s", want, got)
		}
	}
	if got := highlight("if (x) { run(y) }", "javascript"); !strings.Contains(got, `<span class="si-tk-k">if</span> (`) || !strings.Contains(got, `<span class="si-tk-f">run</span>(`) {
		t.Errorf("a keyword before ( stays a keyword: %s", got)
	}
	if got := highlight("ls --all", "bash"); !strings.Contains(got, `<span class="si-tk-f"> --all</span>`) {
		t.Errorf("bash flags: %s", got)
	}
	for _, c := range cases {
		if got := highlight(c.code, c.lang); !strings.Contains(got, c.want) {
			t.Errorf("%s: %q\n got %s\nwant %s", c.lang, c.code, got, c.want)
		}
	}
}
