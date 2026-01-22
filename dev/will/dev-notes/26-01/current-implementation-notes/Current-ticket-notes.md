# Notes

## 29 seems underscoped

- If you are going to go to the effort of formatting the learnings
  - Why go to the effort of doing this without catching more than simply the sharp edges?
  - And why only markdown? That seems completely retarded when I am engineering a systems-architecture-reviewer on top of it.
  - Thoughts are that this needs to be a hybrid system - dual jsonl with a query schema optimised specifically for agentic query in a context lite manner
    - All well and good but how do I break it up so its not horrendously large files
      - Weekly aggregation -> weekly reviews
        - Once reviewed capture and actioned -> ARCHIVE
  - Where am I capturing user decision inputs - how am I storing responses - user intent etc?

## 38b

- Is .yaml the best format for these config files for agents ?vscode-terminal:/9bd88a2418060bb7b26321ab55c82eed/3
  - Is it GO-native? Is there a better format that is more machine readble?

## 38d

- This whole series kinda infers that the daemon watches for end states...but that it can also monitor logs as they happen and inject blocking responses as agents operate...? Am i reading this wrong?
  - Or is it that its an Endresponse trigger?

## 39

- Might need increased scoping consdidering the refactor we did on 028.
  - Want this to have a metric shit tonne of unit tests, integration tests, edge case tests, fallbacks, db schema tests, db look up tests. Need variety.

## 64

- Could actually probably go with a series of subtickets to really flesh out the Class specific responses/additional context
  - Also really need to make sure that this bit plugs in properly with my memory archival so I can query the logic and decisions of subagents
    - Logic here is that this memory capture can inform systematic, deterministric & data-driven evolution of subagent schemas over time to ensure that expertise is actually gained, not merely simulated.

## 65

- Here would be where the above needs to also be rescoped accordingly to accomodate
  - Stats will need to be readjusted too
- Would probably also like to track the thinking budgets used for each step
  - If some look overly complex maybe opus really is a better escalation cost-benefit wise?
- I'd actually like to track spawned_by and route_chain (epic-parent-child) relationships
- Task type would probably also be useful - in scope, out of scope?
  - Who evaluates this?
    - Heuristic could be something like orchestrator type does something implementation
      - Out of scope flag
        - Routing error? idk if thats helpful

## 69

- Is sprintf-ing the model REALLY the best way to attention gate a model? Sycophantic wandering - this feels like its a wishy washy way of document theatre in a prompt inection way.
  - There has to be some kind of programmatic way of doing the same thing but "whip cracking" them to do the right thing?
    - Code injection somehow?
      - Maybe I should delegate this to opus for a research task
- Again there is yaml - what if i decide something is better.................
  - Might need to rethink this sooner rather than later

## 70

- Again this could probably use additional scoping compliant with the session-archive schema - more data is better.

## 71

- Tests fleshed out for any additional scope

## 75

- Think would be good to add systems architect here
  - Or in fact, rather than make it specific subagents, maybe make it class so that it is agent type agnostic

## 76

- Okay this is fucking CRITICAL as this is the orchestrator - slave handler for spawning background tasks and then setting observable state to _awaiting workers_ or some shit
  - This would be a fantastic way to track work crews dispatched to do xyz
    - Problem
      - Dispatch overseers
        - Overseers assess problem, call in work team
          - Work team work, overseers marshal and broadcast state
            - State can be observed on a worker by worker basis
              - Poll worker - implementing, testing, validating, done?

## 77

- Seriously wtf is a blocking response
  - Does this inject a "stop work" order if something is flagged?
    - What are the flags?

## 78

- Do we capture all these levels of information in our db schema?
  - Like - orchestrator sucks ass at actually managing background agents and getting them to do shit
    - Or, architect is great at plannign but the critical evaluation is they should always be attached to a task of higher complexity as it is a waste of budget to assign them to trivial work crews?

## 80

- So now we are up to polling the events for write/edit on claude.md
  - Just a polling service, should be fit for purpose but will need to check.

## 81

- Actually fucking love this as it stops sycophantic "lets just edit claude.md and expect it fucking works".
  - Need to make sure that the storage schema matches up with overall use case for this
    - It's a fancy blocker that stops documentation enforcement as a default ostensibly

# 87

- This would probably be the point where we would add a cost calculation script to evaluate cost-benefit - Note that the current costing script is really fucking off - need to actually update it with real values.
  - Would be worth having a "valuable output" boolean here so we could remove redundant subagents or functionality
    - IE haiku sucks ass for doing xyz code lanugage or something would be a post hoc eval off this metric

# 88

- This would be actually fantastic to be able to cross reference against other metadata and eval inefficiencies on either cost, task, functuon etc.
  - Should interrogate this schema for cross-db compatability.

# 91 

- I actually dont know what stop gate is either so this would be really good to know up front

# Integration testin (95-onwards)

- This is probably an orchestrator moment which should look through implementations for each module systematically.
  - I should probably create some kind of MR architectural overview for each module so as to guide this.
    - This needs to be pretty comprehensive as this is the final migratory audit bit pre-MVP
- NB this is a huge fucking document so needs to be heavily chunked out.

# Deployment & Cutover
- Needs the same as the above tbh. Once this is up and running -> then we are ready to build a TUI over the top.