/**
 * Type declarations for marked-terminal
 * This module lacks official TypeScript types
 */

declare module "marked-terminal" {
  import type { MarkedExtension } from "marked";

  interface MarkedTerminalOptions {
    code?: (code: string, language?: string) => string;
    blockquote?: (quote: string) => string;
    html?: (html: string) => string;
    heading?: (text: string, level: number) => string;
    firstHeading?: (text: string, level: number) => string;
    hr?: () => string;
    listitem?: (text: string) => string;
    list?: (body: string, ordered: boolean) => string;
    paragraph?: (text: string) => string;
    strong?: (text: string) => string;
    em?: (text: string) => string;
    codespan?: (text: string) => string;
    del?: (text: string) => string;
    link?: (href: string, title: string, text: string) => string;
    href?: (href: string) => string;
  }

  export function markedTerminal(options?: MarkedTerminalOptions): MarkedExtension;
}
