# --- ADD THIS LINE FIRST ---
import pymupdf.layout 
# ---------------------------
import pymupdf4llm
import pathlib

# Convert the PDF to a list of page chunks
md_text_chunks = pymupdf4llm.to_markdown("lisan-al-gaib-architecture-guide-v1.0.pdf", page_chunks=True)

output_dir = pathlib.Path("markdown_pages")
output_dir.mkdir(exist_ok=True)

for i, page in enumerate(md_text_chunks):
    filename = output_dir / f"page_{i+1}.md"
    filename.write_text(page["text"], encoding="utf-8")
    print(f"Saved {filename}")