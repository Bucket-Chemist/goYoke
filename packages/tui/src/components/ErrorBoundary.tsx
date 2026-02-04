import { Component, ErrorInfo, ReactNode } from "react";
import { Box, Text } from "ink";
import { logger } from "../utils/logger.js";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

/**
 * Error boundary component that catches React errors and prevents crashes.
 *
 * Integrates with logger.ts to capture error details for debugging.
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary>
 *   <YourComponent />
 * </ErrorBoundary>
 * ```
 *
 * With custom fallback:
 * ```tsx
 * <ErrorBoundary fallback={<Text>Custom error message</Text>}>
 *   <YourComponent />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Log error to debug file (if DEBUG=true) and memory buffer
    logger.error("Component error caught by ErrorBoundary", {
      message: error.message,
      stack: error.stack,
      componentStack: info.componentStack,
      errorName: error.name,
    });
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <Box borderStyle="single" borderColor="red" padding={1}>
          <Text color="red">
            Component error: {this.state.error?.message}
          </Text>
        </Box>
      );
    }
    return this.props.children;
  }
}
