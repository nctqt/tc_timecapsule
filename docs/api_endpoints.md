
The **CaseChronicle** API is organized around REST principles. All requests utilize standard JSON request and response bodies, with versioned routes prefixed under `/api/v1`.

## Authentication & Authorization Tiers

The API enforces three distinct security levels handled via custom Go middleware:
1. **Public:** Open endpoints requiring no authentication.
2. **Authenticated (`middlewareAuth`):** Requires a valid JWT Bearer token supplied in the `Authorization` header.
3. **Admin Restricted (`adminMiddleware`):** Requires an administrative user context verified via JWT.

---

## System & Health

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | Public | Root service banner and verification. |
| `GET` | `/api/v1/healthz` | Public | Standard container health check endpoint for orchestration. |

---

## Authentication

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/users/register` | Public | Register a new user profile. |
| `POST` | `/api/v1/users/login` | Public | Authenticate credentials and return a signed JWT. |

---

## Cases & Timelines

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/cases` | Public | Retrieve a paginated list of all investigative cases. |
| `GET` | `/api/v1/cases/{case_id}` | Public | Fetch detailed metadata for a specific case. |
| `GET` | `/api/v1/cases/{case_id}/milestones` | Public | List all historical milestones associated with a case. |
| `GET` | `/api/v1/cases/{case_id}/timeline` | Optional Auth | Fetch the chronological timeline. Supports optional user tracking via middleware. |
| `POST` | `/api/v1/cases` | Admin | Create a new investigative case file. |
| `DELETE` | `/api/v1/cases/{case_id}` | Admin | Delete a specific case and cascade-remove related data. |

---

## Milestones

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/milestones/{milestone_id}` | Public | Retrieve details for a specific milestone. |
| `POST` | `/api/v1/cases/{case_id}/milestones` | Admin | Add a new milestone nested under a specific case. |
| `DELETE` | `/api/v1/milestones/{milestone_id}` | Admin | Delete an individual milestone. |

---

## Video Indexing & Media Management

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/videos` | Admin | Retrieve all indexed media records. |
| `GET` | `/api/v1/videos/{video_id}` | Public | Fetch metadata for a specific indexed video. |
| `POST` | `/api/v1/videos` | Admin | Register a new raw video source for indexing. |
| `POST` | `/api/v1/videos/{video_id}/enrich` | Admin | Trigger AI-driven metadata extraction and content enrichment via external APIs. |
| `GET` | `/api/v1/videos/unlinked` | Admin | List all ingested videos not yet assigned to a timeline milestone. |
| `PATCH` | `/api/v1/videos/{video_id}/category` | Admin | Update the classification category of a video. |
| `PATCH` | `/api/v1/videos/{video_id}/status` | Admin | Update the processing status of an indexed video. |
| `GET` | `/api/v1/milestones/{milestone_id}/videos` | Public | List all video sources linked to a specific milestone. |

---

## Milestone-Video Association

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `PUT` | `/api/v1/milestones/{milestone_id}/videos/{video_id}` | Admin | Link an indexed video source to a specific milestone. |
| `DELETE` | `/api/v1/milestones/{milestone_id}/videos/{video_id}` | Admin | Unlink a video source from a milestone. |

---

## Watch History

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/watch` | Authenticated | Record a user video view or progress position. |
| `GET` | `/api/v1/watch/history` | Authenticated | Retrieve the authenticated user's viewing history. |
