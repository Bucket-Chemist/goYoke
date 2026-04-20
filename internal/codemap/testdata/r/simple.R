library(dplyr)
require(ggplot2)
library(stats)

hello <- function(name) {
    paste("Hello,", name)
}

greet <- function(x, y = "world") {
    sprintf("Hi %s, %s", x, y)
}

compute_sum <- function(a, b) {
    a + b
}
