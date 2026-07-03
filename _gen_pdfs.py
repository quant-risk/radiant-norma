"""Gera TODOS os PDFs — agora com PRODUTO_TESE_ROADMAP."""

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path("/Users/henrique/Downloads/cadocs")
DOCS = [
    ("README.md", "README.pdf", dict(
        badge="MANIFESTO · ÍNDICE",
        title="CADOCs — Banco Central do Brasil",
        subtitle="Mapa completo de leiautes oficiais, instruções, normativos, XSDs, exemplos XML e engenharia reversa de concorrentes (Mitra, Matera, cadoc.ai). Inclui DRSAC 2030 ESG.",
        tipo="Documento de referência",
    )),
    ("ENG_REVERSA.md", "ENG_REVERSA.pdf", dict(
        badge="ENGEHARIA REVERSA · ANÁLISE",
        title="Mitra, Matera & Cia — Onde vencer",
        subtitle="Mapa competitivo, arquitetura proposta, pricing, roadmap, personas e gap analysis objetivo — incluindo o gap ESG/DRSAC 2030.",
        tipo="Análise estratégica",
    )),
    ("PRODUTO_TESE_ROADMAP.md", "PRODUTO_TESE_ROADMAP.pdf", dict(
        badge="PRODUTO · TESE · ROADMAP",
        title="Radiant Sentinel",
        subtitle="Sentinela regulatória pra IF brasileira — tese, personas, GTM, produto por fase, arquitetura, UX, compliance pra IF regulada e roadmap de 18 meses pra construir a plataforma que compete com Mitra/Matera/cadoc.ai.",
        tipo="Estratégia & execução",
    )),
]

CHROME = "/Users/henrique/Library/Caches/ms-playwright/chromium_headless_shell-1228/chrome-headless-shell-mac-arm64/chrome-headless-shell"

CSS = """
@page {
    size: A4;
    margin: 22mm 18mm 22mm 18mm;

    @top-right {
        content: counter(page) " / " counter(pages);
        font-family: -apple-system, "Helvetica Neue", Arial, sans-serif;
        font-size: 9pt;
        color: #64748b;
    }

    @bottom-left {
        content: "Radiant Sentinel · Radiant Risk Solutions (marca da Fortvna)";
        font-family: -apple-system, "Helvetica Neue", Arial, sans-serif;
        font-size: 8pt;
        color: #94a3b8;
    }

    @bottom-right {
        content: "Radiant Sentinel — Sentinela regulatória pra IF";
        font-family: -apple-system, "Helvetica Neue", Arial, sans-serif;
        font-size: 8pt;
        color: #94a3b8;
    }
}

@page :first {
    @top-right { content: ""; }
    @bottom-left { content: ""; }
    @bottom-right { content: ""; }
    margin: 0;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

html, body { margin: 0; padding: 0; }

body {
    font-family: -apple-system, "Helvetica Neue", "Segoe UI", Arial, sans-serif;
    font-size: 10.5pt;
    line-height: 1.55;
    color: #1f2937;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;
    background: #ffffff;
}

.cover {
    width: 210mm; height: 297mm;
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 60%, #334155 100%);
    color: #f8fafc;
    padding: 35mm 28mm 14mm 28mm;
    page-break-after: always;
    display: flex; flex-direction: column; justify-content: space-between;
    position: relative; overflow: hidden;
}

.cover::before {
    content: "";
    position: absolute;
    top: -50mm; right: -50mm;
    width: 200mm; height: 200mm;
    background: radial-gradient(circle, rgba(59, 130, 246, 0.15), transparent 70%);
    border-radius: 50%;
}

.cover .accent {
    width: 60pt; height: 5pt;
    background: linear-gradient(90deg, #3b82f6, #8b5cf6);
    margin-bottom: 12mm; border-radius: 3pt;
    position: relative; z-index: 1;
}

.cover .badge {
    display: inline-block;
    background: rgba(59, 130, 246, 0.18);
    border: 1px solid rgba(147, 197, 253, 0.4);
    color: #bfdbfe;
    font-size: 8.5pt; font-weight: 700;
    letter-spacing: 0.15em; text-transform: uppercase;
    padding: 4pt 12pt; border-radius: 100pt;
    margin-bottom: 8mm;
    position: relative; z-index: 1;
    width: fit-content;
}

.cover h1.title {
    font-size: 38pt; font-weight: 800;
    line-height: 1.05; letter-spacing: -0.025em;
    margin: 0 0 10mm 0; color: #ffffff;
    position: relative; z-index: 1;
}
.cover h1.title::before { content: none; }

.cover .subtitle {
    font-size: 13pt; line-height: 1.5;
    font-weight: 400; color: #cbd5e1;
    margin: 0 0 auto 0;
    max-width: 145mm;
    position: relative; z-index: 1;
}

.cover .meta {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 5mm 12mm;
    border-top: 1px solid rgba(148, 163, 184, 0.25);
    padding-top: 8mm;
    position: relative; z-index: 1;
}

.cover .meta-item .label {
    color: #94a3b8;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    font-size: 7pt; font-weight: 700;
    margin-bottom: 1.5pt;
}

.cover .meta-item .value {
        color: #f1f5f9;
        font-weight: 600;
        font-size: 10.5pt;
    }

    .footer-brand {
        font-size: 8.5pt;
        color: #94a3b8;
        font-style: italic;
        margin-top: 2mm;
    }

h1 {
    font-size: 22pt; font-weight: 800;
    color: #0f172a; margin: 0 0 8pt 0;
    letter-spacing: -0.02em;
    page-break-before: always;
}
h1:first-of-type { page-break-before: avoid; }
h1::before {
    content: "";
    display: inline-block;
    width: 36pt; height: 4pt;
    background: linear-gradient(90deg, #3b82f6, #8b5cf6);
    margin-right: 14pt; vertical-align: middle;
}

h2 {
    font-size: 16pt; font-weight: 700;
    color: #0f172a;
    margin: 18pt 0 6pt 0;
    padding: 6pt 0 4pt 0;
    border-bottom: 2px solid #e2e8f0;
    page-break-after: avoid;
}

h3 {
    font-size: 12.5pt; font-weight: 700;
    color: #1e293b;
    margin: 14pt 0 5pt 0;
    page-break-after: avoid;
}

h4 {
    font-size: 11pt; font-weight: 600;
    color: #334155;
    margin: 10pt 0 4pt 0;
    page-break-after: avoid;
}

p { margin: 0 0 7pt 0; }
strong, b { font-weight: 700; color: #0f172a; }
em, i { color: #475569; }

ul, ol { margin: 5pt 0 9pt 0; padding-left: 20pt; }
li { margin-bottom: 3pt; }

blockquote {
    margin: 9pt 0;
    padding: 8pt 14pt;
    border-left: 4px solid #3b82f6;
    background: #eff6ff;
    color: #1e40af;
    border-radius: 0 4pt 4pt 0;
    font-style: italic;
}
blockquote p { margin: 0; }

code {
    font-family: "SF Mono", "Monaco", "Menlo", "Consolas", monospace;
    font-size: 9pt; background: #f1f5f9; color: #be185d;
    padding: 1pt 4pt; border-radius: 3pt;
    word-break: break-word;
}

pre {
    font-family: "SF Mono", "Monaco", "Menlo", "Consolas", monospace;
    font-size: 9pt; line-height: 1.5;
    background: #0f172a; color: #e2e8f0;
    padding: 10pt 12pt; border-radius: 6pt;
    overflow: auto;
    margin: 7pt 0 11pt 0;
    page-break-inside: avoid;
    border: 1px solid #1e293b;
}
pre code { background: transparent; color: inherit; padding: 0; font-size: inherit; }

table {
    width: 100%; border-collapse: collapse;
    margin: 10pt 0;
    page-break-inside: avoid;
    font-size: 9.5pt; background: #ffffff;
    box-shadow: 0 1pt 3pt rgba(0,0,0,0.05);
}

thead { background: #1e293b; color: #f8fafc; }
th {
    padding: 7pt 9pt; text-align: left;
    font-weight: 700; font-size: 9pt;
    letter-spacing: 0.04em; text-transform: uppercase;
    border-bottom: 2px solid #334155;
}
td {
    padding: 6pt 9pt;
    border-bottom: 1px solid #e5e7eb;
    vertical-align: top;
}
tbody tr:nth-child(even) { background: #f8fafc; }
tbody tr:last-child td { border-bottom: 2px solid #cbd5e1; }

hr {
    border: 0; border-top: 1px solid #e5e7eb;
    margin: 18pt 0;
}

a {
    color: #2563eb; text-decoration: none;
    border-bottom: 1px solid rgba(37, 99, 235, 0.3);
    word-break: break-word;
}

img { max-width: 100%; height: auto; border-radius: 4pt; }

ul li input[type="checkbox"] {
    margin-right: 6pt;
}
"""


def md_to_html(md_path: Path, html_path: Path, cover: dict):
    css_tmp = ROOT / "_pdf_style.css"
    css_tmp.write_text(CSS, encoding="utf-8")
    cmd = [
        "pandoc", str(md_path),
        f"--output={html_path}",
        "--from=gfm+smart+footnotes",
        "--to=html5",
        "--wrap=none",
        "--standalone",
        "--syntax-highlighting=none",
        f"--css={css_tmp}",
        "--metadata=lang:pt-BR",
    ]
    print(f"  → pandoc {md_path.name}")
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        print(f"  ERRO: {r.stderr}", file=sys.stderr)
        return False

    html = html_path.read_text(encoding="utf-8")
    html = re.sub(r"<header[^>]*>.*?</header>\s*", "", html, count=1, flags=re.DOTALL)

    cover_html = f"""<div class="cover">
  <div>
    <div class="accent"></div>
    <div class="badge">{cover['badge']}</div>
    <h1 class="title">{cover['title']}</h1>
    <p class="subtitle">{cover['subtitle']}</p>
    <div class="footer-brand">Sob a égide da Radiant, sua IF na norma.</div>
  </div>
  <div class="meta">
    <div class="meta-item">
      <div class="label">Produto</div>
      <div class="value">Radiant Sentinel</div>
    </div>
    <div class="meta-item">
      <div class="label">Tipo</div>
      <div class="value">{cover['tipo']}</div>
    </div>
    <div class="meta-item">
      <div class="label">Versão</div>
      <div class="value">2.0 · 2026-07-03</div>
    </div>
    <div class="meta-item">
      <div class="label">Marca</div>
      <div class="value">Radiant Risk Solutions · Fortvna</div>
    </div>
  </div>
</div>
"""
    html = re.sub(r"(<body[^>]*>)", r"\1\n" + cover_html, html, count=1)
    html_path.write_text(html, encoding="utf-8")
    print(f"  ✓ HTML ({html_path.stat().st_size:,} bytes)")
    return True


def html_to_pdf_chromium(html_path: Path, pdf_path: Path):
    cmd = [
        CHROME, "--headless", "--no-sandbox", "--disable-gpu",
        "--disable-dev-shm-usage", "--no-pdf-header-footer",
        f"--print-to-pdf={pdf_path}",
        f"file://{html_path.absolute()}",
    ]
    print(f"  → chromium → {pdf_path.name}")
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if r.returncode != 0:
        print(f"  ERRO: {r.stderr}", file=sys.stderr)
        return False
    print(f"  ✓ {pdf_path.stat().st_size:,} bytes")
    return True


def main():
    print("=" * 70)
    print("PDFs — README + ENG_REVERSA + PRODUTO_TESE_ROADMAP")
    print("=" * 70)

    pdfs = []
    for md_name, pdf_name, cover in DOCS:
        md = ROOT / md_name
        html = md.parent / f"_build_{md.stem}.html"
        pdf = ROOT / pdf_name

        print(f"\n[{md_name}]")
        ok = md_to_html(md, html, cover) and html_to_pdf_chromium(html, pdf)
        if ok:
            pdfs.append(pdf)
            if html.exists():
                html.unlink()

    print("\n" + "=" * 70)
    print("✓ Concluído")
    for p in pdfs:
        print(f"  • {p}")


if __name__ == "__main__":
    main()
