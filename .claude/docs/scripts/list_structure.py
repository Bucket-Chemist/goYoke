import os
import re

def get_page_number(filename):
    match = re.search(r'page_(\d+)\.md', filename)
    return int(match.group(1)) if match else 0

files = [f for f in os.listdir("markdown_pages") if f.endswith(".md")]
files.sort(key=get_page_number)

for f in files:
    with open(os.path.join("markdown_pages", f), "r") as file:
        lines = file.readlines()
        for line in lines:
            if "#" in line and ("Part " in line or "Appendix" in line):
                     print(f"{f}: {line.strip()}")
