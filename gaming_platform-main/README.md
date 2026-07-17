# Gaming Platform Frontend

React + Vite starter frontend for the Go backend.

## Run

Keep the backend running on port 8080, then in this folder:

```powershell
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

The Vite proxy sends `/api/*` requests to `http://localhost:8080/*`, so you do not need to add CORS to the Go server for local development.
