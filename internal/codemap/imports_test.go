package codemap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRustImports(t *testing.T) {
	src := []byte(`
use std::collections::HashMap;
use crate::utils::helper;
use serde::Deserialize;
use core::fmt;
`)
	graph := extractRustImports(src, "myapp")
	assert.Contains(t, graph.Stdlib, "std")
	assert.Contains(t, graph.Stdlib, "core")
	assert.Contains(t, graph.Internal, "crate")
	assert.Contains(t, graph.External, "serde")
}

func TestExtractRustImports_ProjectName(t *testing.T) {
	src := []byte(`use myapp::config;`)
	graph := extractRustImports(src, "myapp")
	assert.Contains(t, graph.Internal, "myapp")
}

func TestExtractTypeScriptImports(t *testing.T) {
	src := []byte(`
import React from 'react';
import { readFileSync } from 'fs';
import { helper } from './utils';
import type { Config } from '../config';
import path from 'path';
`)
	graph := extractTypeScriptImports(src)
	assert.Contains(t, graph.External, "react")
	assert.Contains(t, graph.Stdlib, "fs")
	assert.Contains(t, graph.Stdlib, "path")
	assert.Contains(t, graph.Internal, "./utils")
	assert.Contains(t, graph.Internal, "../config")
}

func TestExtractTypeScriptImports_ScopedPackage(t *testing.T) {
	src := []byte(`import { something } from '@scope/package/deep/path';`)
	graph := extractTypeScriptImports(src)
	assert.Contains(t, graph.External, "@scope/package")
}

func TestExtractTypeScriptImports_NodePrefix(t *testing.T) {
	src := []byte(`import { readFile } from 'node:fs';`)
	graph := extractTypeScriptImports(src)
	assert.Contains(t, graph.Stdlib, "node:fs")
}

func TestExtractPythonImports(t *testing.T) {
	src := []byte(`
import os
import sys
import pandas
from . import utils
from myapp.core import handler
from collections import OrderedDict
`)
	graph := extractPythonImports(src, "myapp")
	assert.Contains(t, graph.Stdlib, "os")
	assert.Contains(t, graph.Stdlib, "sys")
	assert.Contains(t, graph.Stdlib, "collections")
	assert.Contains(t, graph.External, "pandas")
	assert.Contains(t, graph.Internal, ".")
	assert.Contains(t, graph.Internal, "myapp.core")
}

func TestExtractPythonImports_MultiImport(t *testing.T) {
	src := []byte(`import os, sys, pathlib`)
	graph := extractPythonImports(src, "")
	assert.Contains(t, graph.Stdlib, "os")
	assert.Contains(t, graph.Stdlib, "sys")
	assert.Contains(t, graph.Stdlib, "pathlib")
}

func TestExtractRImports(t *testing.T) {
	src := []byte(`
library(dplyr)
library(stats)
require(ggplot2)
library("methods")
`)
	graph := extractRImports(src)
	assert.Contains(t, graph.External, "dplyr")
	assert.Contains(t, graph.Stdlib, "stats")
	assert.Contains(t, graph.External, "ggplot2")
	assert.Contains(t, graph.Stdlib, "methods")
}

func TestExtractImports_Dispatch(t *testing.T) {
	// Verify ExtractImports routes to the correct extractor.
	rSrc := []byte(`library(dplyr)`)
	graph := ExtractImports(rSrc, "r", "")
	assert.Contains(t, graph.External, "dplyr")

	pySrc := []byte(`import pandas`)
	graph = ExtractImports(pySrc, "python", "")
	assert.Contains(t, graph.External, "pandas")

	unknownSrc := []byte(`#nothing`)
	graph = ExtractImports(unknownSrc, "cobol", "")
	assert.Empty(t, graph.Internal)
	assert.Empty(t, graph.External)
	assert.Empty(t, graph.Stdlib)
}

// --- Fixture-based import classification tests ---

func TestExtractImports_RustFixture(t *testing.T) {
	src, err := os.ReadFile("testdata/rust/simple.rs")
	require.NoError(t, err)
	graph := ExtractImports(src, "rust", "myproject")
	assert.Contains(t, graph.Stdlib, "std", "std:: should be stdlib")
	assert.Contains(t, graph.Internal, "crate", "crate:: should be internal")
	assert.Contains(t, graph.External, "serde", "serde should be external")
}

func TestExtractImports_TypeScriptFixture(t *testing.T) {
	src, err := os.ReadFile("testdata/typescript/simple.ts")
	require.NoError(t, err)
	graph := ExtractImports(src, "typescript", "")
	assert.Contains(t, graph.Stdlib, "fs", "fs should be stdlib (Node built-in)")
	assert.Contains(t, graph.Internal, "./local", "./local should be internal")
	assert.Contains(t, graph.External, "express", "express should be external")
}

func TestExtractImports_PythonFixture(t *testing.T) {
	src, err := os.ReadFile("testdata/python/simple.py")
	require.NoError(t, err)
	graph := ExtractImports(src, "python", "myapp")
	assert.Contains(t, graph.Stdlib, "os", "os should be stdlib")
	assert.Contains(t, graph.Stdlib, "collections", "collections should be stdlib")
}

func TestExtractImports_RFixture(t *testing.T) {
	src, err := os.ReadFile("testdata/r/simple.R")
	require.NoError(t, err)
	graph := ExtractImports(src, "r", "")
	assert.Contains(t, graph.External, "dplyr", "dplyr should be external")
	assert.Contains(t, graph.External, "ggplot2", "ggplot2 should be external")
	assert.Contains(t, graph.Stdlib, "stats", "stats should be stdlib (R base package)")
}
