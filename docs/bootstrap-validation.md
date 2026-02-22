# Bootstrap Validation

## Local Bootstrap and Build Commands

```bash
make build
```

## Runtime Validation Steps

1. Start service:

```bash
make run
```

2. Verify root dashboard serving:

```bash
curl -i http://127.0.0.1:8080/
```

Pass criteria: `200 OK` and HTML content.

3. Verify health endpoint:

```bash
curl -i http://127.0.0.1:8080/api/health
```

Pass criteria: `200 OK` with JSON `{"status":"healthy"}`.

## Safe Bootstrap Constraints

- Bootstrap mode exposes only health and dashboard routes.
- No shell execution path is enabled by default.
- Keep secrets outside source control and inject via environment.
