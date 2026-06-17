---
title: Improve search relevance
description: Sample plan — rank exact title matches above body matches.
tags: [type:plan, status:in-progress, topic:architecture]
type: plan
status: in-progress
---

# Plan — Improve search relevance

> A **plan** is written *before* a non-trivial task and closed with an
> `Outcome` afterwards. This is a sample so you can see the shape.

## Goal

When a query matches a note's title, that note should rank above notes
that only match in the body.

## Context

Search runs over the [[demo/memory/glossary#FTS5]] index described in
[[demo/memory/architecture]]. Today every match is weighted equally.

## Steps

1. Add a title-boost to the FTS5 ranking expression.
2. Verify with a query that hits both a title and a body
   (e.g. `architecture`).
3. Record the choice as a new ADR in [[demo/memory/decisions]].

## Outcome

_To be filled in when the task is done — commit hash, surprises, and any
side findings discovered along the way._

← Back to [[demo/README]]
