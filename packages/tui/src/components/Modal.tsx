/**
 * Modal overlay component - renders centered over content
 * Handles keyboard input (Escape to cancel) and delegates to specific modal types
 */

import React from "react";
import { Box, useInput } from "ink";
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

interface ModalOverlayProps {
  request: ModalRequest;
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
 * ModalOverlay renders a centered modal over the main content
 * - Captures keyboard focus
 * - Escape cancels modal
 * - Delegates to specific modal type for rendering and Enter handling
 */
export function ModalOverlay({ request }: ModalOverlayProps): JSX.Element {
  const { dequeue, cancel } = useStore();

  useInput((input, key) => {
    if (key.escape) {
      cancel(request.id);
    }
  });

  return (
    <Box
      position="absolute"
      width="100%"
      height="100%"
      justifyContent="center"
      alignItems="center"
    >
      <Box
        borderStyle={borders.modal}
        borderColor={colors.warning}
        padding={2}
        flexDirection="column"
      >
        <CurrentModal
          request={request}
          onComplete={(response) => dequeue(request.id, response)}
        />
      </Box>
    </Box>
  );
}
