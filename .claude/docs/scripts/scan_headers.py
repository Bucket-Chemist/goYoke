from __future__ import annotations
import os
import re
from pathlib import Path

def extract_number(path: Path) -> int:
    """Extracts the numerical part from filenames like 'page_12.md'."""
    match = re.search(r'page_(\d+)\.md', path.name)
    if match:
        return int(match.group(1))
    return 0

def scan_headers() -> None:
    directory = Path('markdown_pages')
    if not directory.is_dir():
        print(f"Error: Directory '{directory}' not found.")
        return

    # Get all matching files and sort them numerically
    files = sorted(directory.glob('page_*.md'), key=extract_number)

    for file_path in files:
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                lines = f.readlines()
            
            # Print filename
            print(f"File: {file_path.name}")
            
            # Find and print relevant headers
            for line in lines:
                line = line.strip()
                if (line.startswith('#') or line.startswith('##')) and \
                   ('Part ' in line or 'Appendix' in line):
                    print(line)
                    
        except Exception as e:
            print(f"Error reading {file_path}: {e}")

if __name__ == "__main__":
    scan_headers()
