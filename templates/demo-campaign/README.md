# Demo Campaign

A campaign template used as a test subject for capsule's multi-task pipeline.

## Overview

This project defines a `Contact` type with name, email, phone, and address fields. It deliberately omits validation and formatting functions, which serve as feature gaps for campaign mode testing.

The bead hierarchy spans an epic with two features and four tasks, exercising campaign dispatch across multiple work items.

## Building

```bash
cd src
go build ./...
```

## Purpose

This template is used by `setup-template.sh` to create a fresh, deterministic starting state for capsule campaign test runs. The bead fixtures define tasks across two feature streams (input validation and contact formatting), enabling multi-task orchestration testing.
