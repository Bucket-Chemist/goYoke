/**
 * Modal overlay component - renders centered over content
 * Handles keyboard input (Escape to cancel) and delegates to specific modal types
 */

import React, { Component, ErrorInfo, ReactNode } from "react";
import { Box, Text, useInput } from "ink";
import { useStore } from "../store/index.js";
import type {
  ModalRequest,
  ModalResponse,
  AskPayload,
  ConfirmPayload,
  InputPayload,
  SelectPayload,
} from "../store/slices/modal.js";
import { colors, borders } from "../config/theme.js";
import { AskModal } from "./modals/AskModal.js";
import { ConfirmModal } from "./modals/ConfirmModal.js";
import { InputModal } from "./modals/InputModal.js";
import { SelectModal } from "./modals/SelectModal.js";
import { logger } from "../utils/logger.js";

interface ModalOverlayProps {
  request: ModalRequest;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  onDismiss: () => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Error boundary for modal rendering - catches errors to prevent app crash
 * Displays error message and allows dismissal with Escape key
 */
class ModalErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  override state: ErrorBoundaryState = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo): void {
    void logger.error("Modal rendering error", {
      error: error.message,
      stack: error.stack,
      componentStack: info.componentStack,
    });
  }

  override render(): ReactNode {
    if (this.state.hasError) {
      return <ErrorFallback error={this.state.error} onDismiss={this.props.onDismiss} />;
    }
    return this.props.children;
  }
}

/**
 * Error fallback UI - allows dismissal with Escape
 */
function ErrorFallback({ error, onDismiss }: { error: Error | null; onDismiss: () => void }): JSX.Element {
  useInput((input, key) => {
    if (key.escape) {
      onDismiss();
    }
  });

  return (
    <Box flexDirection="column" padding={1}>
      <Text color="red">Modal Error: {error?.message ?? "Unknown error"}</Text>
      <Text dimColor>Press Escape to dismiss</Text>
    </Box>
  );
}

/**
 * Routes to the appropriate modal component based on request type
 */
function CurrentModal({
  request,
  onComplete,
}: {
  request: ModalRequest;
  onComplete: (response: ModalResponse) => void;
}): JSX.Element {
  switch (request.type) {
    case "ask":
      return <AskModal request={request as ModalRequest<AskPayload>} onComplete={onComplete} />;
    case "confirm":
      return <ConfirmModal request={request as ModalRequest<ConfirmPayload>} onComplete={onComplete} />;
    case "input":
      return <InputModal request={request as ModalRequest<InputPayload>} onComplete={onComplete} />;
    case "select":
      return <SelectModal request={request as ModalRequest<SelectPayload>} onComplete={onComplete} />;
    default:
      return <Box>Unknown modal type</Box>;
  }
}

/**
 * ModalOverlay renders a centered modal as a full content-area replacement.
 * By NOT using position="absolute", the underlying panels are fully unmounted,
 * giving 100% opacity — no bleed-through of background content.
 *
 * Layout.tsx swaps the content area for <ModalOverlay> when modalQueue is non-empty.
 */
export function ModalOverlay({ request }: ModalOverlayProps): JSX.Element {
  const { dequeue, cancel } = useStore();

  useInput((input, key) => {
    if (key.escape) {
      cancel(request.id);
    }
  });

  const modalTitle =
    request.type === "confirm" ? "Confirmation Required" :
    request.type === "select"  ? "Select an Option" :
    request.type === "input"   ? "Input Required" :
                                 "Question";

  return (
    <Box flexGrow={1} flexDirection="column" justifyContent="center" alignItems="center">
      <Box
        borderStyle={borders.modal}
        borderColor={colors.warning}
        paddingX={3}
        paddingY={1}
        flexDirection="column"
        minWidth={52}
      >
        {/* Modal header */}
        <Box marginBottom={1}>
          <Text bold color={colors.warning}>⚡ {modalTitle}</Text>
        </Box>

        <ModalErrorBoundary onDismiss={() => cancel(request.id)}>
          <CurrentModal
            request={request}
            onComplete={(response) => dequeue(request.id, response)}
          />
        </ModalErrorBoundary>
      </Box>
    </Box>
  );
}
