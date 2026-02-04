#!/bin/bash
#
# Terminal Compatibility Manual Testing Script
#
# Run this script in each terminal emulator to verify TUI behavior.
# Results should be documented in docs/terminal-compatibility.md

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║         Terminal Compatibility Test - GOgent TUI            ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "Terminal Environment:"
echo "  TERM:           $TERM"
echo "  COLORTERM:      ${COLORTERM:-not set}"
echo "  Terminal Size:  $(tput cols)x$(tput lines)"
echo ""
echo "══════════════════════════════════════════════════════════════"
echo ""
echo "STARTING TUI APPLICATION..."
echo "When the app launches, verify the following checklist:"
echo ""
echo "Visual Rendering:"
echo "  [ ] Colors render correctly (not washed out or wrong)"
echo "  [ ] Borders render (single, double, or rounded styles)"
echo "  [ ] Text styles work (bold, dim, inverse)"
echo "  [ ] Unicode characters display correctly"
echo "  [ ] No visual artifacts or corruption"
echo ""
echo "Interactive Features:"
echo "  [ ] Resize terminal - layout adjusts smoothly"
echo "  [ ] Tab key switches focus between panels"
echo "  [ ] Text input works in input fields"
echo "  [ ] Arrow keys navigate (if applicable)"
echo "  [ ] Enter key activates selections"
echo "  [ ] Ctrl+C exits cleanly"
echo ""
echo "Error Handling:"
echo "  [ ] No crashes during normal use"
echo "  [ ] Errors display in red bordered boxes"
echo "  [ ] App remains responsive after errors"
echo ""
echo "══════════════════════════════════════════════════════════════"
echo ""
echo "Press ENTER to launch the TUI..."
read -r

cd "$(dirname "$0")/.." || exit 1

if [ ! -f "package.json" ]; then
  echo "ERROR: package.json not found. Are you in the right directory?"
  exit 1
fi

# Launch the TUI
npm run dev

echo ""
echo "══════════════════════════════════════════════════════════════"
echo "Test complete. Document your results in:"
echo "  docs/terminal-compatibility.md"
echo ""
echo "Include:"
echo "  - Terminal name and version"
echo "  - Full/Partial/Not Supported classification"
echo "  - Any issues observed"
echo "  - Degraded features (if partial support)"
echo "══════════════════════════════════════════════════════════════"
