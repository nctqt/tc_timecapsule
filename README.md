# CaseChronicle

An automated true crime investigative timeline application designed to correlate and index historical YouTube broadcasts, news footage, and archival records into a unified chronological interface.

---

## Overview

The goal is to be able to watch a true crime case 'as it occured', following a timeline of major events and watching news reports and speculative commentary videos before the whole story was known. CaseChronicle is a full-stack portfolio project built to solve a specific data problem: indexing and synchronizing fragmented true crime media by exact event timelines. The system ingests multi-source data, processes it through a robust backend service, and presents a responsive dashboard for deep-dive investigations. This is my capstone project for the boot.dev backend engineering course.

---

## Tech Stack

### Backend
* **Language:** Go
* **Database & Querying:** PostgreSQL 16, SQLC, Goose (Migrations)
* **API & Routing:** Standard library
* **Integrations:** YouTube Data API, OpenRouter API

### Frontend
* **Framework:** React, Vite, TypeScript
* **Styling:** Tailwind CSS / CSS Modules
* **State & Data Fetching:** Native Fetch API

---

## Architecture & Design

* **Type-Safe SQL:** Utilizes `sqlc` to generate type-safe Go code from raw SQL queries, eliminating runtime query errors.
* **Containerized Infrastructure:** Runs entirely inside Docker containers (`postgres:16-alpine`) for zero-friction local deployment.
* **RESTful API Design:** Clean separation of concerns with a decoupled backend service and Single Page Application (SPA) frontend.

---
