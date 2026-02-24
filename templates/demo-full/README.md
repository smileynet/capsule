# Demo Full

An extended campaign template for capsule's multi-epic pipeline testing.

## Overview

This project defines a `Board` and `Task` type for a simple task management library. It deliberately omits CRUD, validation, serialization, and statistics functions, which serve as feature gaps for campaign mode testing.

## Hierarchy

The bead hierarchy spans 3 epics, 4 features, and 11 tasks across two priority levels, plus a backlog epic for parking-lot items:

```
demo-1 (epic P0: Core Task Management)
  demo-1.1 (feature: Task CRUD)
    demo-1.1.1 (task: Create task)
    demo-1.1.2 (task: List tasks with filtering)
    demo-1.1.3 (task: Update task status)
  demo-1.2 (feature: Task Validation)
    demo-1.2.1 (task: Validate task title)
    demo-1.2.2 (task: Validate priority range)
demo-2 (epic P1: Persistence & Serialization)
  demo-2.1 (feature: JSON Serialization)
    demo-2.1.1 (task: Marshal board to JSON)
    demo-2.1.2 (task: Unmarshal board from JSON) <- blocked by demo-2.1.1
  demo-2.2 (feature: Board Statistics)
    demo-2.2.1 (task: Count tasks by status)
    demo-2.2.2 (task: Completion percentage) <- blocked by demo-2.2.1
demo-100 (epic P4: Retrospective & Parking Lot)
  demo-100.1 (task: Consider database backend)
  demo-100.2 (task: Evaluate web UI framework)
```

## Building

```bash
cd src
go build ./...
```

## Purpose

This template is used by `setup-template.sh` to create a fresh, deterministic starting state for multi-epic campaign test runs. The broader hierarchy (compared to demo-campaign) exercises cross-epic dispatch, blocking dependencies, and priority ordering.
