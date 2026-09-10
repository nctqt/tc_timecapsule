## Getting Started & Installation

### Prerequisites
* [Docker & Docker Compose](https://www.docs.docker.com/)
* [Go (v1.22+)](https://golang.org/)
* [Node.js (v18+)](https://nodejs.org/)

### Clone the Repositories
```bash
git clone [https://github.com/nctqt/casechronicle.git]
git clone [https://github.com/nctqt/casechronicle_frontend.git]
```

### Environment
```
.env file for backend:
POSTGRES_USER= choose a postgres user
POSTGRES_PASSWORD= choose a postgres password
POSTGRES_PORT=5432
POSTGRES_DB= name your database
DATABASE_URL=postgresql://user:password@localhost:5432/database_name?sslmode=disable
HTTP_PORT=8080
HTTP_HOST=localhost
YT_API_KEY= from youtube.com
OPENROUTER_API_KEY= from openrouter.ai (this uses the free version)
JWT_SECRET= generate a jwt secret
```

### Run
In backend folder:
```
go run ./cmd/api
```
It should say running on port 8080.

In frontend folder:
```
npm run dev
```
It should say running on localhost:5173.
