import React from "react";
import { render } from "ink-testing-library";
import { describe, it } from "vitest";
import { TextInputCore } from "../../src/components/primitives/TextInputCore.js";

describe("Debug: inspect raw frames", () => {
  it("prints frame for empty+focused", () => {
    const { lastFrame } = render(<TextInputCore value="" onChange={() => {}} focused />);
    console.log("empty+focused frame:", JSON.stringify(lastFrame()));
  });

  it("prints frame for value=hello focused", () => {
    const { lastFrame } = render(<TextInputCore value="hello" onChange={() => {}} focused />);
    console.log("hello+focused frame:", JSON.stringify(lastFrame()));
  });

  it("prints frame for empty+unfocused+placeholder", () => {
    const { lastFrame } = render(<TextInputCore value="" onChange={() => {}} placeholder="Type here" focused={false} />);
    console.log("empty+unfocused+placeholder frame:", JSON.stringify(lastFrame()));
  });

  it("prints after stdin.write 'a'", () => {
    let captured = "";
    const { lastFrame, stdin } = render(<TextInputCore value="" onChange={(v) => { captured = v; }} focused />);
    stdin.write("a");
    console.log("onChange captured:", JSON.stringify(captured));
    console.log("frame after 'a':", JSON.stringify(lastFrame()));
  });

  it("prints after enter write", () => {
    let submitted = false;
    const { stdin } = render(<TextInputCore value="" onChange={() => {}} onSubmit={() => { submitted = true; }} focused />);
    stdin.write("\r");
    console.log("submitted:", submitted);
  });
});
