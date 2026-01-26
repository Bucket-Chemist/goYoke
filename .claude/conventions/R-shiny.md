---
paths:
  - "**/app.R"
  - "**/ui.R"
  - "**/server.R"
  - "**/R/mod_*.R"
  - "**/www/**"
---

# Shiny Application Development Standards

This document provides guidelines for building production-grade Shiny applications. These conventions extend the core R rules and MUST be followed for all Shiny projects.

---

## Core Architectural Principles

### 1. Module-Based Architecture

All non-trivial Shiny apps MUST use modular architecture:

- **Applets/Modules**: Self-contained UI + Server pairs that encapsulate functionality
- **Single Responsibility**: Each module handles one feature or workflow step
- **Composability**: Modules can be nested and combined

```r
# Module UI - takes id, returns tagList
myModuleUI <- function(id) {
    ns <- shiny::NS(id)
    shiny::tagList(
        shiny::textInput(ns("input"), "Enter value")
        , shiny::actionButton(ns("submit"), "Submit")
    )
}

# Module Server - uses moduleServer pattern
myModuleServer <- function(id, shared_data) {
    shiny::moduleServer(id, function(input, output, session) {
        ns <- session$ns
        # Module logic here
    })
}
```

### 2. R6/S4 Hybrid State Management (Recommended Pattern)

For complex apps with undo/revert functionality, use R6 as a state tracker for S4 objects:

**Division of Labor:**
- **S4 Classes**: Represent all core data structures; handle validation, transformation, computation
- **R6 Class (StateManager)**: Track snapshots of S4 objects; enable undo/revert; NO transformation logic

```r
# R6 State Manager
StateManager <- R6::R6Class("StateManager",
    public = list(
        states = list()

        , saveState = function(state_name, s4_object, config = NULL, description = "") {
            self$states[[state_name]] <- list(
                data = s4_object
                , config = config
                , description = description
                , timestamp = Sys.time()
            )
            invisible(self)
        }

        , getState = function(state_name) {
            if (!state_name %in% names(self$states)) {
                stop("State '", state_name, "' not found")
            }
            self$states[[state_name]]$data
        }

        , listStates = function() {
            names(self$states)
        }
    )
)

# Usage in server
state_manager <- StateManager$new()
state_manager$saveState("after_load", my_s4_object, description = "Initial data")
# Later: revert with state_manager$getState("after_load")
```

**Anti-Pattern**: Do NOT rewrite S4 methods in R6 or store data in new R6 formats. The hybrid model leverages strengths of both systems.

### 3. Centralized Reactive Data Flow

Use a single `reactiveValues` object as the central data bus:

```r
# In main server
workflow_data <- shiny::reactiveValues(
    data_raw = NULL
    , data_processed = NULL
    , config = list()
    , state_manager = StateManager$new()
    , tab_status = list()
)

# Pass to modules
myModuleServer("module_id", workflow_data)
```

**Rules:**
- Modules MUST NOT use the global environment for data sharing
- All shared state flows through `workflow_data`
- Each module reads from and writes to `workflow_data`

---

## Module Construction Standards

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Module UI Function | `moduleNameUI` | `qualityControlUI` |
| Module Server Function | `moduleNameServer` | `qualityControlServer` |
| Module ID | snake_case | `"quality_control"` |

### Namespace Handling

**CRITICAL**: Always use namespaced IDs in modules:

```r
# UI: Use ns() for ALL input/output IDs
myModuleUI <- function(id) {
    ns <- shiny::NS(id)
    shiny::tagList(
        shiny::textInput(ns("user_input"), "Label")  # CORRECT
        # shiny::textInput("user_input", "Label")    # WRONG - not namespaced
    )
}

# Server: Use session$ns for dynamic UI
myModuleServer <- function(id, workflow_data) {
    shiny::moduleServer(id, function(input, output, session) {
        ns <- session$ns

        output$dynamic_ui <- shiny::renderUI({
            shiny::selectInput(ns("dynamic_select"), "Choose", choices = c("A", "B"))
        })
    })
}
```

### Return Patterns

Modules that compute artifacts MUST return them as reactives:

```r
designMatrixServer <- function(id, workflow_data) {
    shiny::moduleServer(id, function(input, output, session) {
        # Compute design matrix
        design_matrix <- shiny::reactive({
            shiny::req(workflow_data$data_raw)
            build_design_matrix(workflow_data$data_raw, input$factors)
        })

        # Return the reactive for parent to use
        return(design_matrix)
    })
}

# Parent server
design_result <- designMatrixServer("design", workflow_data)
shiny::observe({
    workflow_data$design_matrix <- design_result()
})
```

---

## UI/UX Conventions

### Layout Structure

- **Main Sections**: Wrap major applet UIs in `shiny::wellPanel()`
- **Tabbed Content**: Use `shiny::tabsetPanel()` for sub-steps within a workflow stage
- **Inputs/Actions**: Group related inputs and action buttons in nested `wellPanel` (left/top)
- **Outputs**: Display plots, tables in the main panel area

```r
myModuleUI <- function(id) {
    ns <- shiny::NS(id)
    shiny::wellPanel(
        shiny::fluidRow(
            shiny::column(3,
                shiny::wellPanel(
                    shiny::selectInput(ns("method"), "Method", choices = c("A", "B"))
                    , shiny::actionButton(ns("run"), "Run Analysis")
                )
            )
            , shiny::column(9,
                shiny::plotOutput(ns("main_plot"), height = "600px")
            )
        )
    )
}
```

### File/Directory Selection

**ALWAYS** use `shinyFiles` for file and directory inputs:

```r
# UI
shinyFiles::shinyFilesButton(
    ns("file_select")
    , label = "Select File"
    , title = "Choose a file"
    , multiple = FALSE
)

# Server
volumes <- c(Home = fs::path_home(), getVolumes()())
shinyFiles::shinyFileChoose(input, "file_select", roots = volumes)

shiny::observeEvent(input$file_select, {
    file_path <- shinyFiles::parseFilePaths(volumes, input$file_select)$datapath
    # Use file_path
})
```

**NEVER** use base `shiny::fileInput()` for production apps - it lacks native file system access.

### Resizable Plots

For complex plots, use `shinyjqui::jqui_resizable()`:

```r
shiny::column(9,
    shinyjqui::jqui_resizable(
        shiny::plotOutput(ns("my_plot"), height = "600px", width = "100%")
    )
)
```

**MUST** define initial `height` and `width` to prevent collapsing.

### Explicit Namespaces

**MUST** use explicit package prefixes for all Shiny functions:

```r
# CORRECT
shiny::fluidRow(
    shiny::column(6, shiny::textInput(ns("x"), "X"))
    , shiny::column(6, shiny::actionButton(ns("go"), "Go"))
)

# WRONG - Will fail if packages are detached/reloaded
fluidRow(
    column(6, textInput(ns("x"), "X"))
)
```

---

## Testing with shinytest2

### Setup

```r
# Install
install.packages("shinytest2")

# Create test file
usethis::use_test("app")
```

### Snapshot Testing

```r
library(shinytest2)

test_that("App launches and basic interaction works", {
    app <- AppDriver$new(app_dir = ".", name = "basic_test")

    # Take initial snapshot
    app$expect_screenshot()

    # Interact with app
    app$set_inputs(method = "B")
    app$click("run")

    # Wait for computation
    app$wait_for_idle()

    # Verify output
    app$expect_screenshot()

    app$stop()
})
```

### Testing Modules in Isolation

```r
test_that("Module server logic works", {
    shiny::testServer(myModuleServer, args = list(workflow_data = mock_data), {
        # Set input
        session$setInputs(method = "A")

        # Trigger action
        session$setInputs(run = 1)

        # Check output
        expect_equal(output$result, expected_value)
    })
})
```

---

## Debugging

### reactlog for Reactive Dependencies

Enable reactive logging to visualize dependency graphs:

```r
# Before running app
options(shiny.reactlog = TRUE)

# Run app, then press Ctrl+F3 (Cmd+F3 on Mac) to open reactlog
shiny::runApp()

# Or programmatically
reactlog::reactlog_show()
```

### Logger Bug Workaround

**CRITICAL**: The `logger` package has issues with `{}` interpolation in reactive contexts:

```r
# BAD - Will cause error in tryCatch or reactive
tryCatch({
    risky_operation()
}, error = function(e) {
    logger::log_error("Error: {e$message}")  # FAILS
})

# GOOD - Use paste() in reactive/error contexts
tryCatch({
    risky_operation()
}, error = function(e) {
    logger::log_error(paste("Error:", e$message))  # SAFE
})
```

### Debug Print Pattern

```r
# Enable verbose mode via option
options(myapp.debug = TRUE)

debug_log <- function(...) {
    if (isTRUE(getOption("myapp.debug"))) {
        message("[DEBUG] ", ...)
    }
}

# Use in server
debug_log("Processing started, n_rows:", nrow(data))
```

---

## Performance

### bindCache for Expensive Computations

Cache reactive results based on inputs:

```r
expensive_result <- shiny::reactive({
    shiny::req(input$dataset, input$method)
    perform_expensive_computation(input$dataset, input$method)
}) |>
    shiny::bindCache(input$dataset, input$method)
```

### Async with promises/future

For long-running operations, use async to avoid blocking:

```r
library(promises)
library(future)
plan(multisession)

observeEvent(input$run_analysis, {
    # Show loading state
    output$status <- renderText("Processing...")

    future_promise({
        # Long-running computation (runs in separate R process)
        heavy_computation(workflow_data$data_raw)
    }) %...>% (function(result) {
        # Update UI with result (back in main process)
        workflow_data$result <- result
        output$status <- renderText("Complete!")
    }) %...!% (function(error) {
        # Handle errors
        output$status <- renderText(paste("Error:", error$message))
    })
})
```

### Plot Caching

Cache rendered plots:

```r
output$main_plot <- shiny::renderPlot({
    shiny::req(workflow_data$data_processed)
    create_complex_plot(workflow_data$data_processed)
}) |>
    shiny::bindCache(workflow_data$data_processed)
```

### Throttle/Debounce Reactive Inputs

Limit how often reactives fire:

```r
# Debounce: Wait for input to settle (good for text input)
search_debounced <- shiny::debounce(reactive(input$search_text), 500)

# Throttle: Limit frequency (good for sliders)
slider_throttled <- shiny::throttle(reactive(input$slider), 250)
```

---

## Error Handling

### validate/need Pattern

Provide user-friendly error messages:

```r
output$analysis_result <- shiny::renderPlot({
    shiny::validate(
        shiny::need(input$dataset, "Please select a dataset")
        , shiny::need(nrow(workflow_data$data_raw) > 0, "Dataset is empty")
        , shiny::need(input$method %in% c("A", "B", "C"), "Invalid method selected")
    )

    # Only runs if all validations pass
    create_plot(workflow_data$data_raw, input$method)
})
```

### req() for Silent Validation

Use `req()` when you want to silently stop execution without error:

```r
output$plot <- shiny::renderPlot({
    shiny::req(workflow_data$data_processed)  # Silently waits if NULL
    shiny::req(input$show_plot)               # Silently waits if FALSE

    create_plot(workflow_data$data_processed)
})
```

### safeError for User-Facing Errors

Wrap errors that should be shown to users:

```r
output$result <- shiny::renderTable({
    tryCatch({
        process_data(input$file)
    }, error = function(e) {
        stop(shiny::safeError(paste("Could not process file:", e$message)))
    })
})
```

---

## Security

### Input Sanitization

**NEVER** trust user input directly:

```r
# BAD - SQL injection risk
query <- paste0("SELECT * FROM users WHERE name = '", input$username, "'")

# GOOD - Use parameterized queries
query <- DBI::sqlInterpolate(conn, "SELECT * FROM users WHERE name = ?", input$username)
```

### XSS Prevention in renderUI

**NEVER** use `htmlOutput` with unsanitized user input:

```r
# BAD - XSS vulnerability
output$user_content <- shiny::renderUI({
    shiny::HTML(input$user_text)  # User could inject <script> tags
})

# GOOD - Escape HTML
output$user_content <- shiny::renderUI({
    shiny::tags$p(input$user_text)  # Automatically escaped
})

# GOOD - Explicit sanitization if HTML is needed
output$user_content <- shiny::renderUI({
    shiny::HTML(htmltools::htmlEscape(input$user_text))
})
```

### Session Security

```r
# Access session info
session$clientData$url_hostname
session$clientData$url_protocol

# Clean up on session end
session$onSessionEnded(function() {
    # Close database connections
    # Clean up temp files
    # Log session end
})
```

---

## Accessibility

### ARIA Labels

Add accessibility labels to interactive elements:

```r
shiny::actionButton(
    ns("submit")
    , "Submit"
    , `aria-label` = "Submit the form"
)

shiny::tags$div(
    role = "region"
    , `aria-labelledby` = ns("section_title")
    , shiny::h3(id = ns("section_title"), "Results")
    , shiny::tableOutput(ns("results_table"))
)
```

### Keyboard Navigation

Ensure interactive elements are keyboard-accessible:

```r
# Use standard HTML elements that are naturally keyboard-accessible
shiny::actionButton()  # Focusable, activates with Enter/Space
shiny::selectInput()   # Arrow key navigation

# For custom interactive elements, add tabindex
shiny::tags$div(
    tabindex = "0"
    , role = "button"
    , `aria-pressed` = "false"
    , onclick = sprintf("Shiny.setInputValue('%s', true)", ns("custom_btn"))
    , onkeydown = "if(event.key === 'Enter') this.click()"
    , "Custom Button"
)
```

---

## Bookmarking

Enable state serialization for shareable URLs:

### Enable Bookmarking

```r
# In UI
shiny::bookmarkButton()

# In server
shiny::enableBookmarking(store = "url")  # or "server" for complex state
```

### Custom Bookmark State

```r
# Exclude certain inputs from bookmarking
shiny::setBookmarkExclude(c("password", "temp_input"))

# Add custom state
shiny::onBookmark(function(state) {
    state$values$custom_data <- workflow_data$processed
})

shiny::onRestore(function(state) {
    workflow_data$processed <- state$values$custom_data
})
```

---

## Shiny Gotchas (CRITICAL)

### Grid Plot Rendering

| Problem | Solution |
|---------|----------|
| `arrangeGrob()` errors with "cannot open Rplots.pdf" | Wrap with `pdf(NULL)` ... `dev.off()` |
| `grid.arrange()` draws immediately, nothing stored | Use `arrangeGrob()` to create grob object |
| Grob doesn't appear in `renderPlot()` | Use `grid::grid.draw(grob)`, not `print()` |

### Dynamic UI Race Conditions

**NEVER** use `renderUI()` to generate output IDs dynamically—causes timing mismatches where outputs bind to non-existent elements.

**MUST** use slot-based static IDs:
```r
# WRONG - dynamic IDs from data
output_id <- paste0("plot_", assay_name)  # Race condition

# CORRECT - static slot IDs, resolve names inside render function
output$plot_assay1 <- renderImage({ ... })  # Bind at startup
output$plot_assay2 <- renderImage({ ... })
```

**MUST** populate reactive values in the SAME function that generates dependent content—not in separate `observe()` blocks.

### Data Type Preservation

**Matrix roundtrip coerces ID columns to character.** Capture type BEFORE conversion, restore AFTER:
```r
original_type <- class(df[[id_col]])[1]
# ... matrix operations ...
if (original_type %in% c("numeric", "integer")) {
    result[[id_col]] <- as.numeric(result[[id_col]])
}
```

**NEVER** use `sapply(df, is.numeric)` to detect sample columns—grabs metadata columns too. Pass explicit column names from design matrix.

### S4 File Integrity

After editing large S4 class files, **MUST** verify no truncation: `wc -l R/func_*_s4_objects.R`. Restore from git if line count drops unexpectedly.

---

## Anti-Patterns to Avoid

| Anti-Pattern | Correct Approach |
|--------------|------------------|
| Global variables for state | Use `reactiveValues` passed to modules |
| `library()` in app code | Use `package::function()` or `@importFrom` |
| `source()` for modules | Use proper module pattern |
| Relative file paths | Use `here::here()` or proper resource management |
| Blocking long operations | Use `future`/`promises` for async |
| `print()` for debugging | Use `logger` or structured debug functions |
| `fileInput()` for production | Use `shinyFiles` for native file access |
| Direct slot access (`@`) | Use accessor methods for S4 objects |

---

## Before Committing Checklist

- [ ] All modules use `NS(id)` and `session$ns` correctly
- [ ] `reactiveValues` used for shared state (no globals)
- [ ] Explicit package namespaces (`shiny::`, `shinydashboard::`)
- [ ] `validate()`/`need()` for user-facing validation
- [ ] Long operations use async patterns
- [ ] No hardcoded file paths
- [ ] `shinytest2` tests for critical workflows
- [ ] Logger uses `paste()` in reactive/error contexts

---

**Remember:** Modular architecture and proper state management are the foundation of maintainable Shiny apps.
