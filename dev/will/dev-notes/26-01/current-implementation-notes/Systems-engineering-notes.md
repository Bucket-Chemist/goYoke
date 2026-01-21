# On efficiency

## Any and all refactors of upcoming ticket series need to be demarcated for option to parallelise across multiple agentic instances

## Can we start nesting obsidian into the /ticket skill

    - Ticket lookup -> search for cognate obsidian dev notes and thoughts -> if(exists) -> check complexity note and route to appropriate subagent for analysis.

## Implmentation skill

    - Similar to above - can we /implementation explore <dev-note.md> where the <> is /path/to the relevent obsidian implementation thought template (or folder)
        - Where we then systematically construct a high level overview of what needs doing with Einstein or Newton (Einstein clone with sonnet model force but unlimited thinking budget and larger context window)
            - Use template to guide ticket generation
                - Capture decisions back into the obsidian notes -> allow user to further think after ticket gen
    - Outcome is a structured plan, ticketed thoughtfully, with a MR points_to type figure like a .mmp

# Data structuring

- Concerns around yaml format to supplement agent guidelines.
  - If there is a more MR format then that should probably be decided sooner rather than later
    - I can frontload the implementation NOW rather than having to spaghetti refactor and break a bunch of dependencies
- Ticket 76-78 would warrant a think about how we capture work crew information to better manage crew type for task
  - Like woodcutter crew for trees, miners for mining
    - Analogous to orchestrator-staffengineer-architect-implementationpro-codereviewer probably being not the right use case for something like a simple grep bug fix.
      - Why does this matter?
        - Cost savings long run, matching crew config for correct use case to get consistent, reliable and performant outputs.

# Benchmarking

## What to benchmark

## How to benchmark it

## What would be standards?

# What the fuck is the point of stop-gate?

- Why did i do this?

## UNDERUTILISED PARALLEL TOOL CALLING

● (Sorry, that was a demonstration attempt that failed - let me just explain!)

How Parallel Tool Calling Works

The key rule: When tool calls are independent (don't depend on each other's results), you can bundle them all in
a single message.

What I Did

Message 1: [11 Edit calls]  
 - Edit 028k.md (4 separate edits)  
 - Edit 028l.md (1 edit)  
 - Edit 028m.md (3 edits)  
 - Edit 028n.md (2 edits)  
 - Edit 028o.md (1 edit)

All 11 edits were independent - none needed the result of another edit to proceed.

Why This Is Fast

Traditional sequential approach (what many agents do):  
 Message 1: Edit 028k (part 1)  
 → Wait for response  
 Message 2: Edit 028k (part 2)  
 → Wait for response  
 Message 3: Edit 028k (part 3)  
 → Wait for response  
 ... (11 round-trips total)

Parallel approach (what I did):  
 Message 1: All 11 edits at once  
 → Single response with all results

Speed difference: ~10-15 seconds vs 60+ seconds for 11 round-trips.

When You CAN'T Parallelize

If edits depend on each other:  
 BAD:

- Read file to see current content
- Edit based on what you read  
  (These MUST be sequential - can't edit before reading)  


GOOD for parallel:

- Edit file A
- Edit file B
- Edit file C  
  (All independent, can batch)  


Why More Agents Don't Do This

1. Harder to reason about - You need to know ALL edits upfront
2. Error handling complexity - If one fails, you get 10 successes + 1 failure
3. Token/context limits - Large parallel calls can hit limits  


But when it works (like this ticket update task where I knew all changes ahead of time), it's dramatically  
 faster.

---

This pattern works for:

- Read (reading multiple files)
- Edit (editing multiple files independently)
- Bash (running independent commands)
- Task (spawning multiple agents in parallel)
- Grep/Glob (searching multiple patterns)  


My system instructions actually encourage this: "You can call multiple tools in a single response. When multiple
independent pieces of information are requested and all commands are likely to succeed, run multiple tool calls
in parallel for optimal performance."

It's just a matter of recognizing when tasks are independent and batching them! 🚀
