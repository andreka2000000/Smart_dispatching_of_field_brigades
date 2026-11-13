# Smart_dispatching_of_field_brigades

Frontend: Next.js App Router + React + TypeScript. Backend: Go (`backend/`).

```sh
npm ci
npm run dev
```

Local frontend: http://127.0.0.1:3000. Design reference: https://dispatch-two-bay.vercel.app/.

- `/` — Russian marketing homepage, independent of the Go API.
- `/app/dispatch` — dispatcher demo; `/app/technician` and `/app/admin` — role workspaces.

- Frontend setup and backend connection: [docs/FRONTEND.md](docs/FRONTEND.md)
- PyVRP evaluation: [docs/PYVRP.md](docs/PYVRP.md)
- Organizer contract: [contracts/api_spec.yaml](contracts/api_spec.yaml)
- Checks: `npm run check`
