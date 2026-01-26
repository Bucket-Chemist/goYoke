---
description: Debug 66 R implementation. Verbose step-trace debugging for R functions, S4/R6 methods, tidyverse pipelines, and Shiny reactives.
globs: ["*.R", "*.r", "*.Rmd", "*.qmd"]
alwaysApply: false
---

# Debug 66 - R Implementation

## Logging Primitives

```r
# Primary logging function - use message() for stderr (doesn't pollute return values)
d66_log <- function(level, ..., .indent = 0) {
    prefix <- paste0("[D66]", strrep("  ", .indent))
    message(sprintf("%s %s", prefix, paste0(...)))
}

# State inspection helper
d66_state <- function(x, name, .indent = 0) {
    prefix <- paste0("[D66]", strrep("  ", .indent))
    message(sprintf("%s DATA: %s | %s | %s", 
        prefix, name, class(x)[1],
        if (is.data.frame(x)) sprintf("%d×%d", nrow(x), ncol(x))
        else if (is.atomic(x)) sprintf("len=%d", length(x))
        else sprintf("len=%d", length(x))
    ))
}

# Timing helper
d66_time <- function() format(Sys.time(), "%H:%M:%OS3")
```

## Instrumentation Patterns

### Function Entry/Exit

```r
my_function <- function(arg1, arg2, ...) {
    # [D66:START] ─────────────────────────
    .d66_start <- Sys.time()
    d66_log("─── ENTER my_function ───────────────────")
    d66_log("  ARG: arg1 =", capture.output(str(arg1, max.level = 1)))
    d66_log("  ARG: arg2 =", capture.output(str(arg2, max.level = 1)))
    on.exit({
        .d66_dur <- round(difftime(Sys.time(), .d66_start, units = "secs"), 3)
        d66_log("─── EXIT my_function (", .d66_dur, "s) ───")
    }, add = TRUE)
    # [D66:END] ───────────────────────────
    
    # ... original function body ...
}
```

### Data Frame State Inspection

```r
# [D66:START] ─────────────────────────
d66_log("  DATA: df before transform")
d66_log("    dims:", nrow(df), "×", ncol(df))
d66_log("    cols:", paste(names(df), collapse = ", "))
d66_log("    NAs:", sum(is.na(df)))
if (nrow(df) > 0) {
    d66_log("    head:")
    message(paste0("[D66]      ", capture.output(print(head(df, 3))), collapse = "\n"))
}
# [D66:END] ───────────────────────────
```

### Tidyverse Pipeline Debugging

For complex `%>%` or `|>` pipelines, use inline inspection:

```r
result <- df %>%
    # [D66:START]
    { d66_log("  STEP: after initial df, rows =", nrow(.)); . } %>%
    # [D66:END]
    filter(x > 0) %>%
    # [D66:START]
    { d66_log("  STEP: after filter, rows =", nrow(.)); . } %>%
    # [D66:END]
    mutate(y = x * 2) %>%
    # [D66:START]
    { d66_log("  STEP: after mutate, cols =", paste(names(.), collapse = ",")); . }
    # [D66:END]
```

### S4 Method Instrumentation

```r
setMethod("process", "MyClass", function(object, ...) {
    # [D66:START] ─────────────────────────
    .d66_start <- Sys.time()
    d66_log("─── ENTER process,MyClass ───────────────────")
    d66_log("  ARG: object slots =", paste(slotNames(object), collapse = ", "))
    on.exit({
        .d66_dur <- round(difftime(Sys.time(), .d66_start, units = "secs"), 3)
        d66_log("─── EXIT process,MyClass (", .d66_dur, "s) ───")
    }, add = TRUE)
    # [D66:END] ───────────────────────────
    
    # ... method body ...
})
```

### R6 Class Methods

```r
MyClass <- R6::R6Class("MyClass",
    public = list(
        process = function(data) {
            # [D66:START] ─────────────────────────
            .d66_start <- Sys.time()
            d66_log("─── ENTER MyClass$process ───────────────────")
            d66_log("  ARG: data =", class(data)[1], "len =", length(data))
            d66_log("  STATE: self$value =", self$value)
            on.exit({
                d66_log("─── EXIT MyClass$process ───")
            }, add = TRUE)
            # [D66:END] ───────────────────────────
            
            # ... method body ...
        }
    )
)
```

### Loop/Map Instrumentation

```r
# purrr::map with progress
results <- purrr::imap(items, function(item, idx) {
    # [D66:START] ─────────────────────────
    d66_log("  ITER: [", idx, "/", length(items), "] processing:", 
            if (is.character(item)) item else class(item)[1])
    # [D66:END] ───────────────────────────
    
    result <- process_item(item)
    
    # [D66:START]
    d66_log("  ITER: [", idx, "] result:", class(result)[1])
    # [D66:END]
    
    result
})

# For large loops, sample iterations
# [D66:START]
.d66_log_interval <- max(1, length(items) %/% 10)  # Log every 10%
# [D66:END]
for (i in seq_along(items)) {
    # [D66:START]
    if (i == 1 || i == length(items) || i %% .d66_log_interval == 0) {
        d66_log("  ITER: [", i, "/", length(items), "]")
    }
    # [D66:END]
}
```

### Conditional Logic

```r
if (condition) {
    # [D66:START]
    d66_log("  BRANCH: condition → TRUE (", deparse(substitute(condition)), ")")
    # [D66:END]
    # ... true branch ...
} else {
    # [D66:START]
    d66_log("  BRANCH: condition → FALSE")
    # [D66:END]
    # ... false branch ...
}
```

### Error Handling (Hybrid)

```r
# [D66:START] ─────────────────────────
tryCatch({
    # [D66:END]
    
    # ... original risky code ...
    
    # [D66:START]
}, error = function(e) {
    d66_log("  ERROR:", conditionMessage(e))
    d66_log("  ERROR STATE: var1 =", capture.output(str(var1)))
    d66_log("  ERROR STATE: var2 =", capture.output(str(var2)))
    stop(e)  # Re-raise with original message
})
# [D66:END] ───────────────────────────
```

## Shiny-Specific Patterns

### Reactive Instrumentation

```r
data_reactive <- reactive({
    # [D66:START]
    d66_log("─── REACTIVE: data_reactive invalidated ───")
    d66_log("  TRIGGER: input$selector =", input$selector)
    .d66_start <- Sys.time()
    # [D66:END]
    
    result <- compute_data(input$selector)
    
    # [D66:START]
    d66_log("  REACTIVE: data_reactive computed in", 
            round(difftime(Sys.time(), .d66_start, units = "secs"), 3), "s")
    d66_log("  REACTIVE: result dims =", nrow(result), "×", ncol(result))
    # [D66:END]
    
    result
})
```

### Observer Debugging

```r
observeEvent(input$button, {
    # [D66:START]
    d66_log("─── OBSERVER: input$button clicked ───")
    d66_log("  STATE: current values =", reactiveValuesToList(input))
    # [D66:END]
    
    # ... observer body ...
})
```

## Statistical/Modeling Extensions

### Model Fitting

```r
# [D66:START]
d66_log("  MODEL: fitting", class(model_spec)[1])
d66_log("  MODEL: formula =", deparse(formula))
d66_log("  MODEL: data dims =", nrow(data), "×", ncol(data))
d66_log("  MODEL: response var summary:")
message(paste0("[D66]    ", capture.output(summary(data[[response_var]])), collapse = "\n"))
# [D66:END]

fit <- fit_model(model_spec, data)

# [D66:START]
d66_log("  MODEL: fit complete")
d66_log("  MODEL: convergence =", fit$converged)
d66_log("  MODEL: coefficients =", length(coef(fit)))
# [D66:END]
```

### Correlation/Matrix Operations

```r
# [D66:START]
d66_log("  MATRIX: computing correlation")
d66_log("  MATRIX: input dims =", nrow(mat), "×", ncol(mat))
d66_log("  MATRIX: NA count =", sum(is.na(mat)))
d66_log("  MATRIX: complete cases =", sum(complete.cases(mat)))
# [D66:END]

cor_mat <- cor(mat, use = "pairwise.complete.obs")

# [D66:START]
d66_log("  MATRIX: result dims =", nrow(cor_mat), "×", ncol(cor_mat))
d66_log("  MATRIX: result range = [", min(cor_mat, na.rm = TRUE), ",", 
        max(cor_mat, na.rm = TRUE), "]")
# [D66:END]
```

## Cleanup

Remove all Debug 66 instrumentation:

```r
# In R, use this to strip D66 lines:
# readLines("file.R") |> 
#   grep("D66", x = _, invert = TRUE, value = TRUE) |>
#   writeLines("file_clean.R")

# Or use sed:
# sed -i '/D66/d' file.R
```

## Quick Copy-Paste Templates

### Minimal Function Wrapper
```r
# [D66:START]
d66_log("─── ENTER %FNAME% ───"); .d66_t <- Sys.time()
on.exit(d66_log("─── EXIT %FNAME% (", difftime(Sys.time(), .d66_t, units="secs"), "s) ───"), add=TRUE)
# [D66:END]
```

### Data Checkpoint
```r
# [D66:START]
d66_log("  DATA: %VAR%", nrow(%VAR%), "×", ncol(%VAR%), "NAs:", sum(is.na(%VAR%)))
# [D66:END]
```
