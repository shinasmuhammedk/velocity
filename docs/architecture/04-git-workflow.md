# Git Workflow

## Branch Strategy

main
 ↑
develop
 ↑
feature/*

## Rules

- main contains stable production code.
- develop contains integration code.
- feature branches contain isolated work.

## Feature Example

feature/auth
feature/orders
feature/logger
feature/matching-engine

## Merge Strategy

feature/*
    ↓
develop
    ↓
main