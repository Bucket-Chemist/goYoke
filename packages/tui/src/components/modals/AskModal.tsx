/**
 * AskModal component - displays question with optional button options
 * Supports Claude Agent SDK AskUserQuestion contract:
 * - header: Short label displayed above the question
 * - options with descriptions
 * - multiSelect mode with Space to toggle checkboxes
 * - Automatic "Other" option for free-text fallback
 * - Escape warns before discarding unsaved free-text
 * - Backward compatible with existing MCP askUser tool
 *
 * Split layout: scrollable content region (top) + fixed options panel (bottom)
 * Options are ALWAYS visible regardless of content length.
 */

import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { ModalRequest, ModalResponse, AskPayload } from "../../store/slices/modal.js";
import { colors } from "../../config/theme.js";
import { TextInput } from "../primitives/TextInput.js";
import { ScrollView } from "../primitives/ScrollView.js";

interface AskModalProps {
  request: ModalRequest<AskPayload>;
  onComplete: (response: ModalResponse) => void;
  onCancel: () => void;
  /** Available terminal rows for scrollable content area (provided by ModalOverlay) */
  contentHeight?: number;
}

export function AskModal({ request, onComplete, onCancel, contentHeight }: AskModalProps): JSX.Element {
  const payload = request.payload as AskPayload;
  const hasOptions = payload.options && payload.options.length > 0;

  // Build options list with automatic "Other" appended
  const allOptions = hasOptions
    ? [...payload.options!, { label: "Other", value: "other", description: "Type a custom answer" }]
    : [];

  const [selectedIndex, setSelectedIndex] = useState(0);
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(new Set());
  const [freeTextValue, setFreeTextValue] = useState(payload.defaultValue || "");
  const [showOtherInput, setShowOtherInput] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const isMultiSelect = payload.multiSelect === true;
  const isOtherOption = (index: number) => hasOptions && index === allOptions.length - 1;

  // Check if user has typed content that differs from initial value
  const hasUnsavedText = freeTextValue.trim() !== (payload.defaultValue || "").trim() &&
                         freeTextValue.trim().length > 0;

  const handleEscape = () => {
    if (hasUnsavedText && !showConfirm) {
      setShowConfirm(true);
    } else {
      onCancel();
    }
  };

  // Handle input for option navigation and selection
  useInput(
    (input, key) => {
      if (showOtherInput) {
        // When in "Other" text input mode, TextInput handles input
        return;
      }

      if (key.escape) {
        handleEscape();
        return;
      }

      if (hasOptions) {
        // Arrow navigation
        if (key.upArrow) {
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : allOptions.length - 1
          );
        } else if (key.downArrow) {
          setSelectedIndex((prev) =>
            prev < allOptions.length - 1 ? prev + 1 : 0
          );
        } else if (input === " " && isMultiSelect) {
          // Space toggles selection in multi-select mode
          setSelectedIndices((prev) => {
            const next = new Set(prev);
            if (next.has(selectedIndex)) {
              next.delete(selectedIndex);
            } else {
              next.add(selectedIndex);
            }
            return next;
          });
        } else if (key.return) {
          // Enter submits selection
          if (isOtherOption(selectedIndex)) {
            // Show text input for "Other" option
            setShowOtherInput(true);
          } else if (isMultiSelect) {
            // Multi-select: return all selected labels joined with ", "
            if (selectedIndices.size === 0) {
              // If nothing selected, select current highlighted option
              const option = allOptions[selectedIndex];
              if (option) {
                onComplete({ type: "ask", value: option.label });
              }
            } else {
              const selectedLabels = Array.from(selectedIndices)
                .sort((a, b) => a - b)
                .map((idx) => allOptions[idx])
                .filter((opt): opt is NonNullable<typeof opt> => opt !== undefined)
                .map((opt) => opt.label);
              onComplete({ type: "ask", value: selectedLabels.join(", ") });
            }
          } else {
            // Single-select: return the selected option's label
            const option = allOptions[selectedIndex];
            if (option) {
              onComplete({ type: "ask", value: option.label });
            }
          }
        }
      }
    },
    { isActive: hasOptions && !showOtherInput && !showConfirm }
  );

  // Handle Escape in free-text mode
  useInput(
    (input, key) => {
      if (key.escape) {
        handleEscape();
      }
    },
    { isActive: (showOtherInput || !hasOptions) && !showConfirm }
  );

  // Handle confirmation dialog input
  useInput((input, key) => {
    if (key.return || input.toLowerCase() === "y") {
      onCancel();
    } else if (input.toLowerCase() === "n") {
      setShowConfirm(false);
    }
  }, { isActive: showConfirm });

  const handleFreeTextSubmit = () => {
    onComplete({ type: "ask", value: freeTextValue });
  };

  // Determine help text
  let helpText: string;
  if (showOtherInput || !hasOptions) {
    helpText = "Enter Submit • Esc Cancel";
  } else if (isMultiSelect) {
    helpText = "↑↓ Navigate • Space Toggle • Enter Submit • Esc Cancel";
  } else {
    helpText = contentHeight !== undefined
      ? "↑↓ Navigate • PgUp/PgDn Scroll • Enter Submit • Esc Cancel"
      : "↑↓ Navigate • Enter Select • Esc Cancel";
  }

  if (showConfirm) {
    return (
      <Box flexDirection="column" gap={1}>
        <Text color="yellow">⚠ Discard typed text?</Text>
        <Box marginTop={1}>
          <Text>You have unsaved text: &quot;{freeTextValue}&quot;</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Y Discard • N Continue editing</Text>
        </Box>
      </Box>
    );
  }

  // Content region: header + question text
  const contentRegion = (
    <Box flexDirection="column" gap={1}>
      {payload.header && (
        <Box>
          <Text dimColor>[{payload.header}]</Text>
        </Box>
      )}
      <Text wrap="wrap">{payload.message}</Text>
    </Box>
  );

  // Options region: option list or free text input + help text
  const optionsRegion = (
    <Box flexDirection="column" flexShrink={0}>
      {hasOptions && !showOtherInput ? (
        <Box flexDirection="column">
          {allOptions.map((option, index) => {
            const isSelected = index === selectedIndex;
            const isChecked = selectedIndices.has(index);

            return (
              <Box key={index} flexDirection="column">
                <Box>
                  {isMultiSelect && (
                    <Text color={isSelected ? colors.primary : colors.muted}>
                      {isChecked ? "☑ " : "☐ "}
                    </Text>
                  )}
                  <Text color={isSelected ? colors.primary : colors.muted}>
                    {!isMultiSelect && isSelected ? "▶ " : isMultiSelect ? "" : "  "}
                    {option.label}
                    {option.description && ` — ${option.description}`}
                  </Text>
                </Box>
              </Box>
            );
          })}
        </Box>
      ) : (
        <Box>
          <TextInput
            value={freeTextValue}
            onChange={setFreeTextValue}
            onSubmit={handleFreeTextSubmit}
            placeholder={showOtherInput ? "Type your custom answer..." : "Type your answer..."}
            focused={true}
          />
        </Box>
      )}

      {/* Help text */}
      <Box marginTop={1}>
        <Text dimColor>{helpText}</Text>
      </Box>
    </Box>
  );

  // Split layout: scrollable content (top) + fixed options (bottom)
  if (contentHeight !== undefined && contentHeight > 0) {
    return (
      <Box flexDirection="column">
        {/* TOP: Scrollable content region */}
        <ScrollView
          height={contentHeight}
          autoScroll={false}
          focused={false}
          disableArrowKeys={true}
        >
          {contentRegion}
        </ScrollView>

        {/* Separator */}
        <Box>
          <Text dimColor>{"─".repeat(44)}</Text>
        </Box>

        {/* BOTTOM: Fixed options panel — never clipped */}
        {optionsRegion}
      </Box>
    );
  }

  // Fallback: no split layout (full-screen mode or no height constraint)
  return (
    <Box flexDirection="column" gap={1}>
      {contentRegion}
      {optionsRegion}
    </Box>
  );
}
