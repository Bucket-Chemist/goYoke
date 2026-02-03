# GOfortress TUI

Terminal User Interface for GOfortress built with React Ink.

## Quick Start

```bash
# Development mode (with hot-reload)
npm run dev

# Type check
npm run typecheck

# Build
npm run build

# Run production build
npm start
```

## Development

The TUI is built using:
- **React Ink** - Terminal UI components
- **TypeScript** - Type safety
- **Zustand** - State management (coming soon)
- **tsx** - Fast TypeScript execution with watch mode

### Interactive Spike Demo

When you run `npm run dev`, the app starts in demo mode:

- Press **1** - Hello screen (default)
- Press **2** - Layout spike (2-panel split test)
- Press **3** - Responsive layout (terminal resize test)
- Press **4** - Border styles (single, double, round, bold)
- Press **Ctrl+C** - Exit

### Hot Reload

The development server watches all files in `src/` and automatically reloads on changes:

1. Run `npm run dev`
2. Edit any file in `src/`
3. Save the file
4. See instant updates in terminal

## Theme System

All colors and styles are centralized in `src/config/theme.ts`:

```typescript
import { colors, borders } from "./config/theme.js";

// Use theme constants, don't hardcode colors
<Text color={colors.primary}>Text</Text>
<Box borderColor={colors.focused} borderStyle={borders.panel}>
```

## Project Structure

```
src/
├── index.tsx                    # Entry point
├── App.tsx                      # Root component
├── config/
│   └── theme.ts                # Theme constants
└── components/
    ├── LayoutSpike.tsx         # Layout testing
    ├── ResponsiveLayout.tsx    # Resize handling
    └── BorderStyleTest.tsx     # Border validation
```

## Spike Results

See `.claude/tmp/ink-spike-results.md` for detailed findings from the Ink layout spike (TUI-004).

**Key findings:**
- ✅ Percentage-based layouts work perfectly
- ✅ Terminal resize handled automatically
- ✅ All border styles render without artifacts
- ✅ Hot-reload works flawlessly
- ✅ Theme integration successful

## Next Steps

- [ ] TUI-005: Zustand state management
- [ ] TUI-006: Focus manager integration
- [ ] TUI-007: 3-panel layout implementation
- [ ] TUI-008: MCP client integration
