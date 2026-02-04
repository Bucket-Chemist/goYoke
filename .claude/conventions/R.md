# Agent Guidelines for R Code Quality

This document provides guidelines for maintaining high-quality R code. These rules MUST be followed by all AI coding agents and contributors.

## Core Principles

All code you write MUST be fully optimized.

"Fully optimized" includes:
- Maximizing algorithmic big-O efficiency for memory and runtime
- Using parallelization and vectorization where appropriate
- Following proper style conventions (tidyverse style guide, DRY principle)
- No extra code beyond what is absolutely necessary to solve the problem the user provides (i.e., no technical debt)

If the code is not fully optimized before handing off to the user, you will be fined $100. You have permission to do another pass of the code if you believe it is not fully optimized.

---

## R Version and Modern Features

**MUST** target R 4.1+ for new projects; R 4.3+ preferred.

### R 4.1+ Features to Adopt
- **Native pipe `|>`** instead of magrittr `%>%`
- **Lambda syntax** `\(x) x + 1` instead of `function(x) x + 1`
- **`...names()`** for extracting names from `...` arguments

### R 4.2+ Features to Adopt
- **Improved placeholder** `_` in pipes (with named arguments)
- **`chooseOpsMethod()`** for better S4/S3 method dispatch

### R 4.3+ Features to Adopt
- **`toTitleCase()` improvements** for string handling
- **Native R serialization v3** for better cross-platform compatibility

**Example - Modern R Syntax:**
```r
# PREFERRED: Native pipe with lambda
processed_data <- raw_data |>
    filter(\(x) !is.na(x$value)) |>
    mutate(log_value = log2(value + 1))

# PREFERRED: Lambda in apply functions
results <- lapply(data_list, \(df) {
    df |>
        filter(q_value < 0.05) |>
        arrange(desc(log2fc))
})
```

---

## Preferred Tools

### Package Management
- **MUST** use `renv` for dependency management and reproducibility
- **MUST** commit `renv.lock` to version control
- **MUST** run `renv::snapshot()` after adding/updating packages
- **MUST** use `renv::restore()` to recreate environments
```r
renv::init()            # Initialize renv in project
renv::install("dplyr")  # Install packages
renv::snapshot()        # Lock dependencies
renv::restore()         # Restore from lockfile
```

### Development Tools
- **styler** for code formatting (`styler::style_file()`, `styler::style_dir()`)
- **lintr** for static code analysis
- **roxygen2** for documentation (mandatory for all functions)
- **testthat** for testing with **testthat::test_dir()** for batch execution
- **logger** for structured logging (replaces `print`/`message`)
- **profvis** for profiling before optimization
- **covr** for test coverage analysis

### Data Manipulation Tools
- **dplyr/tidyr** for tidy data manipulation
- **purrr** for functional programming on lists/vectors
- **data.table** for performance-critical operations on very large tables

### Tidy Evaluation with String Variables (CRITICAL)

`{{ }}` is for **symbols passed as function arguments**, NOT string variables. For strings, use:

| Verb | String Variable Pattern |
|------|------------------------|
| `pull()` | `df[[var]]` (base R only) |
| `rename()` | `rename(new = !!rlang::sym(var))` |
| `group_by()` | `group_by(across(all_of(var)))` |
| `filter()`/`mutate()` | `.data[[var]]` works |
| `select()` | `all_of(var)` |

**MUST** namespace-prefix tidyselect helpers in packages: `dplyr::all_of()`, not `all_of()`.

**Default to base R** (`df[[col]]`) for simple extraction—no tidy eval complexity.

---

## dplyr 1.1+ Features

### .by Argument (Preferred over group_by)

The `.by` argument provides inline grouping without needing `ungroup()`:

```r
# PREFERRED (dplyr 1.1+)
df |> summarise(total = sum(value), .by = category)

# LEGACY (still works but more verbose)
df |> group_by(category) |> summarise(total = sum(value)) |> ungroup()

# Multiple grouping variables
df |> summarise(mean_val = mean(value), .by = c(region, year))
```

### reframe() for Multi-Row Results

Use `reframe()` when your summary returns multiple rows per group:

```r
# summarise() now warns for multi-row results; use reframe()
df |> reframe(quantile_df(height), .by = species)

# Example: returning multiple quantiles
calculate_quantiles <- function(x) {
    tibble(
        quantile = c(0.25, 0.50, 0.75)
        , value = quantile(x, c(0.25, 0.50, 0.75))
    )
}

df |> reframe(calculate_quantiles(value), .by = group)
```

### pick() for Column Selection in Mutate/Summarise

Use `pick()` to select columns inside data-masking functions:

```r
# Select columns by pattern
df |> mutate(row_sum = rowSums(pick(starts_with("x"))))

# Multiple tidyselect patterns
df |> summarise(across(pick(where(is.numeric)), mean), .by = category)
```

### join_by() for Flexible Joins

Enhanced join syntax with non-equi joins:

```r
# Equi-join (standard)
transactions |> inner_join(customers, join_by(customer_id))

# Non-equi join (range conditions)
events |> inner_join(
    periods
    , join_by(date >= start_date, date <= end_date)
)

# Closest match join
df |> inner_join(reference, join_by(closest(timestamp >= ref_time)))
```

### R 4.4+ Features

#### Null Coalescing Operator (%||%)

```r
# Returns right side if left is NULL
config$port %||% 8080

# Equivalent to:
if (is.null(config$port)) 8080 else config$port
```

---

### S4 Method Calls (CRITICAL)

**MUST** verify parameter names in `R/allGenerics.R` before calling S4 methods. Common traps:
- Full descriptive names: `ruv_number_k` not `k`
- British spelling: `normalisation_method` not `normalization_method`
- Prefixed names: `itsd_aggregation` not `aggregation`

### UI-to-Function Mapping

UI inputs may control **one** parameter while others need hardcoded values. **MUST** read function docs to identify which parameters are type selectors (hardcode) vs user choices (wire to UI).

### Proteomics/Bioinformatics Tools
- **SummarizedExperiment** as base container for omics data
- **MultiAssayExperiment** for multi-omics integration
- **Biobase** for legacy ExpressionSet compatibility
- **arrow** for out-of-memory data handling

---

## Code Style and Formatting

- **MUST** use meaningful, descriptive variable and function names
- **MUST** follow tidyverse style guide strictly
- **MUST** use 4 spaces for indentation (never tabs)
- **MUST** use `<-` for assignment, `=` only for function arguments
- **NEVER** use emoji or unicode emulating emoji except in tests
- Limit line length to 80-100 characters

### Naming Conventions
| Element | Convention | Example |
|---------|------------|---------|
| Functions | camelCase (verbs) | `normalizeProteomicsData()` |
| Variables | snake_case (nouns) | `sample_annotation` |
| Constants | UPPER_SNAKE_CASE | `MAX_Q_VALUE` |
| S4 Classes | PascalCase | `ProteomicsData` |
| R6 Classes | PascalCase | `ExperimentManager` |

### Leading Comma Convention
**MUST** place commas at the beginning of new lines in multi-line constructs:
```r
# CORRECT: Leading commas
my_list <- list(
    item1 = "apple"
    , item2 = "banana"
    , item3 = "cherry"
)

df |>
    select(
        sample_id
        , protein_id
        , intensity
    ) |>
    filter(intensity > 0)
```

---

## Documentation

### roxygen2 Requirements
- **MUST** include roxygen2 documentation for ALL functions (exported and internal)
- **MUST** document `@param`, `@return`, description for every function
- **MUST** use `@export`, `@examples`, `@importFrom` appropriately

### Critical roxygen2 Rules

**NEVER add roxygen comments to `allGenerics.R`:**
```r
# BAD - Will cause parsing errors
#' @export
setGeneric("normalizeData", function(object, ...) standardGeneric("normalizeData"))

# GOOD - Only bare setGeneric calls
setGeneric("normalizeData", function(object, ...) standardGeneric("normalizeData"))
```

**Tag order (recommended):**
1. `@title` - Brief title
2. `@name` - Explicit topic name (required for S4 methods)
3. Description paragraph(s)
4. `@param` - Parameter documentation
5. `@return` - Return value documentation
6. `@importFrom` - Package imports
7. `@export` - Export directive
8. `@examples` - Usage examples

**Tag Conflicts - `@describeIn` and `@name` are mutually exclusive:**
```r
# BAD - Will cause error: "@describeIn can not be used with @name"
#' @describeIn plotPca Method for MetaboliteAssayData
#' @name plotPca,MetaboliteAssayData-method
#' @export
setMethod(f = "plotPca", ...)

# GOOD - Use @name and @title instead
#' @title Plot PCA for MetaboliteAssayData
#' @name plotPca,MetaboliteAssayData-method
#' @export
setMethod(f = "plotPca", ...)
```

**Inheritance Tags - Referenced topics MUST exist:**
- `@inheritParams` and `@inheritDoc` require the referenced topic to exist
- Verify referenced function/topic is documented before using inheritance
- Consider explicit parameter documentation if reference is unclear

**Duplicate Tags:**
- **Only one `@export` per function/method** - multiple tags cause issues

**S4 Method Documentation:**
```r
#' @title Normalize Between Samples for ProteomicsData
#' @name normaliseBetweenSamples,ProteomicsData-method
#' @param theObject Object of class ProteomicsData
#' @param normalisation_method Method to use for normalization
#' @return Modified ProteomicsData object with normalized values
#' @importFrom limma normalizeCyclicLoess
#' @export
setMethod(
    f = "normaliseBetweenSamples"
    , signature = "ProteomicsData"
    , definition = function(theObject, normalisation_method = NULL) {
        # Implementation
    }
)
```

### Code Commenting Philosophy
- **Explain the "Why", not the "What":** Focus on rationale and scientific reasoning
- **Target audience:** Assume biologist/analyst reader
- Use section dividers for complex functions: `# --- Section Name ---`
- **NEVER** commit commented-out code; delete it

```r
# --- Filter Low Variance Features ---
# Remove features with low variance across samples, as they are less likely
# to be informative for downstream differential analysis or clustering.
# Using IQR as a robust variance measure resistant to outliers.
low_variance_threshold <- 0.1
features_to_keep <- calculateFeatureIQR(data_matrix) > low_variance_threshold
filtered_matrix <- data_matrix[features_to_keep, ]
```

---

## S4 Object-Oriented Programming

### Core S4 Principles
- **MUST** use S4 classes inheriting from `SummarizedExperiment` for omics data
- **MUST** use accessor methods instead of direct slot access (`@`)
- **MUST** implement `setValidity` for all custom classes
- **MUST** use dedicated constructor functions with validation

### S4 Class Hierarchy Pattern
```r
# --- Base Class Definition ---
setClass(
    "QuantitativeOmicsData"
    , contains = "SummarizedExperiment"
    , slots = c(
        processing_log = "list"
        , analysis_params = "list"
    )
)

setValidity("QuantitativeOmicsData", function(object) {
    errors <- character()
    if (!is.list(object@processing_log)) {
        errors <- c(errors, "processing_log must be a list")
    }
    if (length(errors) == 0) TRUE else errors
})

# --- Derived Class ---
setClass(
    "ProteomicsData"
    , contains = "QuantitativeOmicsData"
    , slots = c(
        protein_groups = "character"
        , quantification_method = "character"
    )
)

# --- Constructor Function ---
createProteomicsData <- function(
    assay_matrix
    , col_data
    , row_data
    , quantification_method = "LFQ"
) {
    stopifnot(
        is.matrix(assay_matrix)
        , is.data.frame(col_data) || is(col_data, "DataFrame")
        , nrow(col_data) == ncol(assay_matrix)
    )

    se <- SummarizedExperiment(
        assays = list(intensity = assay_matrix)
        , colData = col_data
        , rowData = row_data
    )

    new(
        "ProteomicsData"
        , se
        , processing_log = list()
        , analysis_params = list()
        , protein_groups = rownames(assay_matrix)
        , quantification_method = quantification_method
    )
}
```

### S4 Generics and Methods Pattern
```r
# In allGenerics.R - NO roxygen comments
setGeneric("normalizeData", function(object, method, ...) {
    standardGeneric("normalizeData")
})

# In methods-normalize.R - Full documentation
#' @title Normalize Data for ProteomicsData
#' @name normalizeData,ProteomicsData-method
#' @param object ProteomicsData object
#' @param method Normalization method: "median", "quantile", "vsn"
#' @return Normalized ProteomicsData object
#' @export
setMethod(
    f = "normalizeData"
    , signature = "ProteomicsData"
    , definition = function(object, method = "median", ...) {
        # Implementation specific to proteomics
        assay_data <- assay(object, "intensity")
        normalized <- switch(
            method
            , median = .normalizeMedian(assay_data)
            , quantile = .normalizeQuantile(assay_data)
            , vsn = .normalizeVsn(assay_data)
            , stop("Unknown method: ", method)
        )
        assay(object, "intensity") <- normalized
        object@processing_log <- c(
            object@processing_log
            , list(list(step = "normalize", method = method, time = Sys.time()))
        )
        object
    }
)
```

### S4 Type Checking (CRITICAL)

When checking if an object is an S4 object, **MUST** use the correct functions:

```r
# CORRECT: Use isS4() to check if something is an S4 object
if (!isS4(my_object)) {
    stop("Expected an S4 object")
}

# CORRECT: Use is() or inherits() to check specific class
if (!methods::is(my_object, "ProteomicsData")) {
    stop("Expected a ProteomicsData object")
}

# WRONG: "S4" is NOT a class name - this will ALWAYS return FALSE
if (!methods::is(my_object, "S4")) {  # BUG! "S4" is not a class
    stop("This check always fails for valid S4 objects")
}
```

**Key distinction:**
- `isS4(x)` - Checks the object's **type system** (is it an S4 object?)
- `methods::is(x, "ClassName")` - Checks **class inheritance** (does it inherit from ClassName?)
- `inherits(x, "ClassName")` - Also checks class inheritance (works for S3 and S4)

The string `"S4"` is not a class that objects inherit from—it's a type system designation. S4 objects inherit from their specific classes (e.g., `"ProteomicsData"`, `"SummarizedExperiment"`), not from `"S4"`.

---

## S7 OOP System (PREFERRED for New Code)

S7 is the R Consortium's new OOP system, designed to supersede S3/S4/R6 for new code.
**Use S7 for all new OOP code in R 4.3+.**

### Defining Classes

```r
library(S7)

# Simple class with typed properties
Person <- new_class("Person",
    properties = list(
        name = class_character,
        age = class_integer
    )
)

# With validation
PositiveNumber <- new_class("PositiveNumber",
    properties = list(
        value = class_double
    ),
    validator = function(self) {
        if (self@value <= 0) "value must be positive"
    }
)

# With inheritance
Employee <- new_class("Employee",
    parent = Person,
    properties = list(
        employee_id = class_character,
        department = class_character
    )
)
```

### Property Access

```r
person <- Person(name = "Alice", age = 30L)
person@name         # "Alice" (getter)
person@age <- 31L   # Setter
```

### Methods

```r
# Define generic
greet <- new_generic("greet", "x")

# Implement method
method(greet, Person) <- function(x) {
    paste0("Hello, ", x@name, "!")
}

greet(person)  # "Hello, Alice!"
```

### When to Use Which OOP System

| System | Use For | Key Characteristic |
|--------|---------|-------------------|
| **S7** | All new OOP code (preferred) | Modern, typed properties, validators |
| S4 | Bioconductor packages requiring S4 | Formal, complex dispatch |
| R6 | Reference semantics, mutable state | Pass-by-reference |
| S3 | Simple method dispatch only | Minimal, no type checking |

---

## R6 State Management

### When to Use R6
- **Use R6 for:** Mutable state, complex workflows, caching, connection management
- **Use S4 for:** Data containers, method dispatch, Bioconductor integration

### R6 Pattern for Large Data Workflows
```r
#' @title Experiment Manager
#' @description R6 class for managing proteomics analysis workflows with caching
#' @export
ExperimentManager <- R6::R6Class(
    "ExperimentManager"
    , public = list(
        #' @field data ProteomicsData object
        data = NULL

        #' @field cache List of cached computations
        , cache = NULL

        #' @description Initialize manager with data
        #' @param proteomics_data ProteomicsData object
        , initialize = function(proteomics_data) {
            stopifnot(is(proteomics_data, "ProteomicsData"))
            self$data <- proteomics_data
            self$cache <- list()
            private$.log_step("initialized")
        }

        #' @description Run CPU-intensive operation with caching
        #' @param operation Character name of operation
        #' @param fn Function to execute
        #' @param ... Arguments passed to fn
        , run_cached = function(operation, fn, ...) {
            cache_key <- digest::digest(list(operation, ...))
            if (!is.null(self$cache[[cache_key]])) {
                logger::log_info("Cache hit for {operation}")
                return(self$cache[[cache_key]])
            }
            logger::log_info("Computing {operation}")
            result <- fn(self$data, ...)
            self$cache[[cache_key]] <- result
            private$.log_step(operation)
            result
        }

        #' @description Clear cache
        , clear_cache = function() {
            self$cache <- list()
            invisible(self)
        }
    )
    , private = list(
        .log = list()

        , .log_step = function(step) {
            private$.log <- c(
                private$.log
                , list(list(step = step, time = Sys.time()))
            )
        }
    )
)
```

---

## Concurrency and Parallelization

### Package Selection
| Workload | Solution |
|----------|----------|
| Independent list operations | `future.apply::future_lapply()` |
| Tidyverse-style parallel map | `furrr::future_map()` |
| Bioconductor parallel | `BiocParallel::bplapply()` |
| CPU-bound matrix ops | `RcppParallel` or `data.table` |
| Very large data chunks | `future` with `multisession` plan |

### future/future.apply Setup
```r
library(future)
library(future.apply)

# Cross-platform parallel backend (MUST use multisession on Windows)
plan(multisession, workers = parallel::detectCores() - 1)

# IMPORTANT: Always set seed for reproducibility
results <- future_lapply(
    data_list
    , process_fn
    , future.seed = TRUE  # MUST set for reproducibility
)

# Cleanup when done (good practice)
plan(sequential)
```

### furrr for Tidyverse Workflows
```r
library(furrr)

# Set plan BEFORE using furrr functions
plan(multisession, workers = 4)

# Parallel map with progress bar
results <- future_map(
    data_list
    , slow_fn
    , .progress = TRUE
    , .options = furrr_options(seed = TRUE)
)

# Parallel map with data frame output
results_df <- future_map_dfr(
    split_data
    , \(chunk) {
        chunk |>
            filter(q_value < 0.05) |>
            summarise(mean_fc = mean(log2fc))
    }
    , .id = "group"
    , .options = furrr_options(seed = TRUE)
)
```

### Wrapper Functions for CPU-Intensive S4 Operations
**MUST** create simple wrappers for parallelizing operations on large S4 objects:

```r
#' @title Parallel Wrapper for CPU-Intensive Operations
#' @description Execute function in parallel across data chunks
#' @param object S4 object (ProteomicsData, etc.)
#' @param fn Function to apply to each chunk
#' @param chunk_by Column in rowData to split by (e.g., "protein_group")
#' @param workers Number of parallel workers
#' @param ... Additional arguments passed to fn
#' @return Combined results
#' @export
parallelApplyS4 <- function(
    object
    , fn
    , chunk_by = NULL
    , workers = parallel::detectCores() - 1
    , ...
) {
    stopifnot(is(object, "SummarizedExperiment"))

    # Set up parallel backend
    oplan <- plan(multisession, workers = workers)
    on.exit(plan(oplan), add = TRUE)

    # Split data into chunks
    if (is.null(chunk_by)) {
        # Split by row indices
        n_rows <- nrow(object)
        chunk_size <- ceiling(n_rows / workers)
        indices <- split(seq_len(n_rows), ceiling(seq_len(n_rows) / chunk_size))
        chunks <- lapply(indices, \(idx) object[idx, ])
    } else {
        # Split by grouping variable
        groups <- rowData(object)[[chunk_by]]
        chunks <- lapply(unique(groups), \(g) object[groups == g, ])
    }

    # Execute in parallel with progress
    logger::log_info("Processing {length(chunks)} chunks across {workers} workers")

    results <- future_lapply(
        chunks
        , fn
        , ...
        , future.seed = TRUE
    )

    results
}

# Usage example for proteomics normalization
normalizeChunkedProteomics <- function(prot_data, method = "vsn") {
    # Wrapper for parallel VSN normalization on large datasets
    results <- parallelApplyS4(
        prot_data
        , fn = \(chunk) {
            # CPU-intensive normalization per chunk
            assay_data <- assay(chunk, "intensity")
            normalized <- vsn::justvsn(assay_data)
            assay(chunk, "intensity") <- normalized
            chunk
        }
        , workers = 4
    )

    # Combine results back into single object
    do.call(rbind, results)
}
```

### Parallel Pattern for Large Matrix Operations
```r
#' @title Parallel Correlation Matrix for Large Proteomics Data
#' @param prot_data ProteomicsData object
#' @param method Correlation method
#' @return Correlation matrix
#' @export
parallelCorrelation <- function(prot_data, method = "pearson") {
    mat <- assay(prot_data, "intensity")
    n_samples <- ncol(mat)

    # Only parallelize if matrix is large enough
    if (n_samples < 50) {
        return(cor(mat, method = method, use = "pairwise.complete.obs"))
    }

    plan(multisession, workers = parallel::detectCores() - 1)
    on.exit(plan(sequential), add = TRUE)

    # Compute correlations in parallel by column blocks
    col_pairs <- combn(seq_len(n_samples), 2, simplify = FALSE)

    cor_values <- future_sapply(
        col_pairs
        , \(pair) cor(mat[, pair[1]], mat[, pair[2]], method = method, use = "complete.obs")
        , future.seed = TRUE
    )

    # Reconstruct symmetric matrix
    cor_mat <- matrix(1, nrow = n_samples, ncol = n_samples)
    for (i in seq_along(col_pairs)) {
        pair <- col_pairs[[i]]
        cor_mat[pair[1], pair[2]] <- cor_values[i]
        cor_mat[pair[2], pair[1]] <- cor_values[i]
    }

    dimnames(cor_mat) <- list(colnames(mat), colnames(mat))
    cor_mat
}
```

### BiocParallel for Bioconductor Workflows
```r
library(BiocParallel)

# Register parallel backend
register(MulticoreParam(workers = 4))  # Unix/Mac
# register(SnowParam(workers = 4))     # Windows

# Use with Bioconductor functions that support BPPARAM
results <- bplapply(
    split_data
    , processChunk
    , BPPARAM = MulticoreParam(workers = 4)
)
```

### Memory-Conscious Parallelization
```r
#' @title Memory-Safe Parallel Processing for Large S4 Objects
#' @description Process large objects in memory-efficient chunks
processLargeObject <- function(object, fn, chunk_size = 1000) {
    n_features <- nrow(object)
    n_chunks <- ceiling(n_features / chunk_size)

    # Use sequential futures to avoid memory duplication
    plan(sequential)

    results <- vector("list", n_chunks)

    for (i in seq_len(n_chunks)) {
        start_idx <- (i - 1) * chunk_size + 1
        end_idx <- min(i * chunk_size, n_features)

        # Process chunk
        chunk <- object[start_idx:end_idx, ]
        results[[i]] <- fn(chunk)

        # Force garbage collection between chunks
        rm(chunk)
        gc()

        logger::log_debug("Processed chunk {i}/{n_chunks}")
    }

    do.call(rbind, results)
}
```

---

## Error Handling

- **NEVER** silently swallow exceptions without logging
- **MUST** use `stopifnot()` for input validation
- **MUST** use `tryCatch()` with specific error handling
- **MUST** log errors with `logger::log_error()`

### Error Handling Pattern
```r
processData <- function(object, method) {
    # Input validation
    stopifnot(
        is(object, "ProteomicsData")
        , is.character(method)
        , length(method) == 1
    )

    tryCatch(
        {
            result <- performAnalysis(object, method)
            logger::log_info("Analysis completed successfully")
            result
        }
        , error = function(e) {
            # NEVER use {} interpolation in logger inside tryCatch
            logger::log_error(paste("Analysis failed:", e$message))
            stop(e)
        }
        , warning = function(w) {
            logger::log_warn(paste("Warning during analysis:", w$message))
            invokeRestart("muffleWarning")
        }
    )
}
```

---

## Function Design

- **MUST** keep functions focused on a single responsibility
- **MUST** keep functions under 50-75 lines
- **NEVER** use mutable default arguments
- Prefer pure functions (no side effects)
- Return early to reduce nesting
- Limit parameters to 5-7; use config objects for more

```r
# GOOD: Early return pattern
validateInput <- function(data, threshold) {
    if (is.null(data)) {
        return(NULL)
    }
    if (!is.numeric(threshold)) {
        stop("threshold must be numeric")
    }
    if (threshold < 0 || threshold > 1) {
        stop("threshold must be between 0 and 1")
    }

    # Main logic only reached with valid inputs
    data[data > threshold]
}
```

---

## Testing (testthat)

### Requirements
- **MUST** use testthat for all testing
- **MUST** create test fixtures with `set.seed()` for reproducibility
- **MUST** test S4 class validity and method dispatch
- **NEVER** delete test files or fixtures

### Test Structure
```r
# tests/testthat/test-normalize.R
test_that("normalizeData median method works correctly", {
    # Arrange
    set.seed(42)
    test_data <- createTestProteomicsData(n_proteins = 100, n_samples = 10)

    # Act
    result <- normalizeData(test_data, method = "median")

    # Assert
    expect_s4_class(result, "ProteomicsData")
    expect_equal(ncol(result), ncol(test_data))
    expect_true(all(!is.na(assay(result, "intensity"))))
})

test_that("normalizeData fails with invalid method", {
    test_data <- createTestProteomicsData()

    expect_error(
        normalizeData(test_data, method = "invalid")
        , regexp = "Unknown method"
    )
})
```

---

## Performance Optimization

### Profiling First
- **MUST** profile before optimizing with `profvis`
- **NEVER** optimize without profiling data
```r
profvis::profvis({
    result <- expensiveOperation(large_data)
})
```

### Optimization Techniques
- Use vectorized operations over loops
- Use matrices for numeric data (avoid data frames in hot paths)
- Use `data.table` for large table operations
- Use `memoise` for caching expensive pure functions
- Use `Rcpp` for critical bottlenecks

### Memory Management
- Remove large unused objects: `rm(large_obj); gc()`
- Monitor memory: `lobstr::obj_size()`, `pryr::mem_used()`
- Use `arrow` for larger-than-memory datasets
- Process in chunks for very large matrices

---

## Reproducibility

### Seed Management
- **MUST** use `set.seed()` before ALL stochastic operations
- Document seed values in analysis parameters
- Use `set.seed()` in test fixtures

### Dependency Management
- **MUST** use `renv` for all projects
- **MUST** commit `renv.lock` to version control
```r
renv::init()      # Initialize
renv::snapshot()  # Lock current state
renv::restore()   # Recreate environment
```

### Parameter Tracking
- Track all analysis parameters in config files or script headers
- Log parameters with results
- Use the `processing_log` slot in S4 objects

### Session Info
- **MUST** save session info with results for reproducibility
```r
# Save with results
session_info <- sessionInfo()
# Or more detailed
session_info <- devtools::session_info()
```

### Version Control
- **MUST** use Git for ALL code, scripts, `renv.lock`, config files
- **MUST** commit often with descriptive messages
- **NEVER** commit `.Rhistory`, `.RData`, or large data files

---

## MultiOmics-Specific Guidelines

### Interoperability
- Maintain consistent sample and feature identifiers across omics layers
- Use stable IDs (e.g., Ensembl gene IDs, UniProt accessions)
- Document ID sources and mappings
- Use standard `colData`/`rowData` column names where applicable

### Integration Tools
- Use `MultiAssayExperiment` for managing linked assays
- Consider `mixOmics` for advanced integration analysis
- Document data merging/joining strategies (inner/outer joins)
- Document rationale for normalization methods

### Missing Values
- **Understand missingness type:** MNAR (Missing Not At Random) vs MAR/MCAR
- Visualize patterns with `visdat`, `naniar`
- Choose and document appropriate imputation methods
- Consider type-specific defaults via S4 methods for `imputeMissingValues`
```r
# Visualize missing data patterns
visdat::vis_miss(assay_df)
naniar::gg_miss_upset(assay_df)
```

---

## Quality Control Standards

### Documentation Requirements
- **MUST** justify all filtering thresholds in comments/logs
- **MUST** document normalization choices and rationale

### Visualization
- Generate plots (PCA, density, boxplots, heatmaps) before/after each major QC step
- Use generic plotting functions with S4 dispatch (e.g., `plotPCA(object, ...)`)

### Impact Tracking
- Log features/samples removed at each step
- Track changes in data distribution
- Use the `processing_log` slot in S4 objects
```r
# Track processing step
object@processing_log <- c(
    object@processing_log
    , list(list(
        step = "filter_low_counts"
        , features_removed = sum(!keep_features)
        , threshold = min_count
        , time = Sys.time()
    ))
)
```

---

## Statistical Best Practices

### Multiple Testing Correction
- **MUST** correct p-values using `p.adjust()` when testing multiple hypotheses
- Prefer method="BH" (Benjamini-Hochberg) for FDR control
- Report BOTH raw and adjusted p-values
```r
results$p_adjusted <- p.adjust(results$p_value, method = "BH")
```

### Effect Sizes
- **MUST** report effect sizes alongside p-values
- Use log2 fold change for expression data
- Use Cohen's d for group comparisons
- Visualize with volcano plots
```r
# Volcano plot pattern
ggplot(results, aes(x = log2fc, y = -log10(p_adjusted))) +
    geom_point(aes(color = significant)) +
    geom_hline(yintercept = -log10(0.05), linetype = "dashed")
```

---

## External Resources and Caching

### API Requests
- **MUST** use `httr2` for robust HTTP requests
- Configure retries, error handling, user-agent, timeout
```r
library(httr2)

response <- request("https://api.example.com/data") |>
    req_retry(max_tries = 3, backoff = ~ 2) |>
    req_timeout(seconds = 30) |>
    req_user_agent("MyPackage/1.0") |>
    req_error(is_error = \(resp) resp_status(resp) >= 400) |>
    req_perform()
```

### Caching Strategies
- Use `memoise` for function-level caching of expensive operations
- Use RDS caching for intermediate results
- Provide cache invalidation mechanisms
```r
library(memoise)

# Memoize expensive function
fetchAnnotations <- memoise(function(ids) {
    # Expensive API call or computation
    biomaRt::getBM(...)
})

# Simple RDS caching pattern
getCachedResult <- function(cache_path, compute_fn, force_refresh = FALSE) {
    if (!force_refresh && file.exists(cache_path)) {
        return(readRDS(cache_path))
    }
    result <- compute_fn()
    saveRDS(result, cache_path)
    result
}
```

---

## Package and Dependency Management

### Loading Packages
- Use `pacman::p_load()` for convenient loading/installation
- Use `conflicted::conflict_prefer()` to resolve namespace conflicts
```r
# Convenient loading with auto-install
pacman::p_load(dplyr, tidyr, ggplot2, SummarizedExperiment)

# Explicit conflict resolution
library(conflicted)
conflict_prefer("filter", "dplyr")
conflict_prefer("select", "dplyr")
conflict_prefer("lag", "dplyr")
```

### Namespace Conflicts
- Be explicit with `package::function()` when conflicts exist
- Document known conflicts in project README
- Use `conflicted` package to force explicit resolution

---

## Security

- **NEVER** store secrets, API keys, or passwords in code
- **NEVER** use `eval(parse(text = user_input))`
- **NEVER** print or log URLs containing API keys
- **MUST** validate all external inputs
- **MUST** add `.env` and credentials files to `.gitignore`

---

## Shiny Application Development

**MUST** use explicit namespaces for all Shiny functions:
```r
# BAD - Will fail if packages are detached/reloaded
tabItem(tabName = "home", h3("Welcome"), br())

# GOOD - Always works
shinydashboard::tabItem(
    tabName = "home"
    , shiny::h3("Welcome")
    , shiny::br()
)
```

### Logger Bug in Reactive Contexts
**NEVER** use `{}` interpolation in logger calls inside error handlers or reactive contexts:
```r
# BAD - Will cause error
log_error("Error: {e$message}")

# GOOD - Safe in all contexts
log_error(paste("Error:", e$message))
```

---

## Logging

### logger Package Setup
- **MUST** use `logger` for structured logging instead of `print`/`message`
- Configure appropriate log levels, appenders, and layout
```r
library(logger)

# Configure logging
log_threshold(INFO)
log_appender(appender_tee(file = "analysis.log"))
log_layout(layout_glue_colors)

# Use appropriate levels
log_debug("Detailed debugging info")
log_info("Processing started")
log_warn("Missing values detected, using defaults")
log_error("Failed to load file")
log_fatal("Unrecoverable error, aborting")
```

### Log Levels
| Level | Use Case |
|-------|----------|
| DEBUG | Detailed tracing, variable values |
| INFO | Normal operation milestones |
| WARN | Unexpected but recoverable situations |
| ERROR | Failures that don't stop execution |
| FATAL | Unrecoverable errors |

### Logger Interpolation Bug (CRITICAL)
The `logger` package has a known issue with string interpolation in certain contexts:
- **NEVER** use `{}` interpolation in `tryCatch` error handlers
- **NEVER** use `{}` interpolation in Shiny reactive contexts
- **ALWAYS** use `paste()` or `sprintf()` in these contexts
```r
# Safe contexts for interpolation:
log_info("Processing {n_samples} samples")  # OK in regular functions

# Unsafe contexts - use paste():
tryCatch(
    { risky_operation() }
    , error = function(e) {
        log_error(paste("Failed:", e$message))  # MUST use paste()
    }
)
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Correct Approach |
|--------------|------------------|
| `df[,1]` (numeric indexing) | `df$col` or `df[["col"]]` |
| `attach()` / `detach()` | Use explicit references |
| `rm(list = ls())` | Never use; restart R session |
| `setwd()` | Use `here::here()` or relative paths |
| Bare `%>%` in packages | Use `\|>` or import from magrittr |
| `source()` for shared code | Create proper packages |
| Deep nesting (>3 levels) | Refactor into smaller functions |

---

## Project Structure

```
myproject/
├── R/
│   ├── allClasses.R          # S4 class definitions
│   ├── allGenerics.R         # setGeneric() calls ONLY
│   ├── methods-normalize.R   # Method implementations
│   └── utils.R               # Helper functions
├── tests/
│   └── testthat/
│       ├── helper-fixtures.R # Test data generators
│       └── test-normalize.R  # Tests
├── man/                      # Generated by roxygen2
├── vignettes/
├── DESCRIPTION
├── NAMESPACE                 # Generated by roxygen2
├── renv.lock                 # Dependency lock file
├── .Rprofile                 # renv activation
└── .gitignore
```

---

## Before Committing Checklist

- [ ] All tests pass (`devtools::test()`)
- [ ] R CMD check passes (`devtools::check()`)
- [ ] Documentation builds (`devtools::document()`)
- [ ] Code formatted (`styler::style_pkg()`)
- [ ] Linter passes (`lintr::lint_package()`)
- [ ] All functions have roxygen2 documentation
- [ ] No commented-out code or debug statements
- [ ] `set.seed()` used before all stochastic operations
- [ ] `renv::snapshot()` if dependencies changed
- [ ] No hardcoded credentials or file paths

---

**Remember:** Prioritize clarity and maintainability over cleverness.
