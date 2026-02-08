# Demo Brownfield

A brownfield template used as a test subject for the capsule pipeline.

## Overview

This project defines a `Contact` type with name, email, and phone fields. It deliberately omits input validation functions, which serve as feature gaps for pipeline testing.

## Building

```bash
cd src
go build ./...
```

## Purpose

This template is used by `setup-template.sh` to create a fresh, deterministic starting state for capsule pipeline test runs. The bead fixtures define tasks to implement the missing validation functions.
