import os
import re
from pathlib import Path

# Configuration
INPUT_DIR = Path("markdown_pages")
OUTPUT_DIR = Path("assembled_sections")

# Roman numeral map for file prefixes
ROMAN_TO_INT = {
    "I": 1, "II": 2, "III": 3, "IV": 4, "V": 5,
    "VI": 6, "VII": 7, "VIII": 8, "IX": 9, "X": 10
}

def get_filename(header_text):
    # header_text expected: "Part <Roman>: <Title>" or similar
    # Clean up markdown if present
    clean_header = header_text.replace('*', '').replace('#', '').strip()
    
    # Extract Roman numeral
    match = re.search(r'Part ([IVX]+)', clean_header)
    if not match:
        # Fallback if pattern doesn't match
        return f"XX_{clean_header.replace(' ', '_')}.md"
    
    roman = match.group(1)
    number = ROMAN_TO_INT.get(roman, 0)
    
    # Format filename
    # Replace ": " with "_" and spaces with "_"
    safe_name = clean_header.replace(': ', '_').replace(':', '').replace(' ', '_')
    # Remove any other potential bad chars
    safe_name = re.sub(r'[^\w\-]', '_', safe_name)
    
    return f"{number:02d}_{safe_name}.md"

def main():
    print("Starting assembly...")
    OUTPUT_DIR.mkdir(exist_ok=True)
    
    # 1. Get pages sorted numerically
    # Use explicit integer conversion for sorting
    try:
        pages = sorted(INPUT_DIR.glob("page_*.md"), key=lambda p: int(p.stem.split('_')[1]))
    except Exception as e:
        print(f"Error sorting pages: {e}")
        return

    print(f"Found {len(pages)} pages.")

    current_content = []
    # Initial header assumption
    current_part_header = "Part I: Executive Preamble" 
    
    # Split pattern: 
    # Matches: _Continue to Part ... [→->]_ # Part ...
    # Group 1: The footer (to be removed)
    # Group 2: The new header (to be kept)
    split_pattern = re.compile(r'(_Continue to .*? (?:→|->)_)\s*(# Part .*)')

    for page in pages:
        try:
            text = page.read_text(encoding='utf-8')
        except Exception as e:
            print(f"Error reading {page}: {e}")
            continue
        
        # Search for split
        match = split_pattern.search(text)
        
        if match:
            # We found a split point
            print(f"Found split in {page.name}")
            
            # pre_split belongs to current_part
            pre_split = text[:match.start()]
            current_content.append(pre_split)
            
            # Write current part
            filename = get_filename(current_part_header)
            out_path = OUTPUT_DIR / filename
            out_path.write_text("".join(current_content), encoding='utf-8')
            print(f"  -> Saved {out_path}")
            
            # Start new part
            raw_new_header = match.group(2)
            # Update header for the NEXT file (clean version)
            current_part_header = raw_new_header.lstrip('# ').strip()
            
            # The content for the new part starts with the header
            post_split = raw_new_header + text[match.end():]
            current_content = [post_split]
            
        else:
            # No split, just append
            current_content.append(text)
            
    # Write the last part
    if current_content:
        filename = get_filename(current_part_header)
        out_path = OUTPUT_DIR / filename
        out_path.write_text("".join(current_content), encoding='utf-8')
        print(f"  -> Saved {out_path} (Final)")

    print("Assembly complete.")

if __name__ == "__main__":
    main()
