# Playbook: Deploying a Self-Updating Daemon with systemd + Private GitHub Releases

Deploy a hestia app as a long-running systemd service that periodically checks a
**private GitHub Releases** repo, stages newer builds, and swaps itself in
place — no SSH, no package manager, no human in the loop.

Read [release-publishing-github.md](release-publishing-github.md) first: it
defines the tag/version contract and the asset-naming rule the daemon depends on.

## Two deployment modes

| | Section A — systemd-native (recommended) | Section B — spawn-and-swap (fallback) |
|---|---|---|
| Apply behavior | copy staged binary over the executable, exit cleanly | spawn the new binary as a child, exit |
| systemd unit | `Restart=always` | `Restart=no` |
| Unit state after update | stays `active` | shows `inactive` (app keeps running) |
| Crash recovery | systemd restarts automatically | needs an external watchdog |
| Requires | hestia with `UPDATE_SYSTEMD=true` (current code) | nothing extra (unmodified pipeline) |

Section A is what you should run. Section B documents deployments that must use
the original spawn-and-swap handoff.

---

## Section A — systemd-native daemon

### 1. Filesystem layout

```
/opt/hestia/hestia            # the running executable (owned by hestia:hestia)
/etc/hestia/hestia.env        # environment (0600, hestia:hestia)
/var/lib/hestia/              # UPDATE_DATA_DIR: staged update + SQLite DB + logs
```

The service user must be able to **write the executable and its directory** —
the swap replaces `/opt/hestia/hestia` in place. Everything else follows the
usual data layout:

- `UPDATE_DATA_DIR` — staged `update` binary and the app's data directory.
- `DB_PATH=/var/lib/hestia/hestia.db`
- `LOG_PATH=/var/lib/hestia/server.log`

Create the user and directories:

```bash
useradd --system --home /var/lib/hestia --shell /usr/sbin/nologin hestia
mkdir -p /opt/hestia /var/lib/hestia /etc/hestia
chown hestia:hestia /opt/hestia /var/lib/hestia /etc/hestia
```

### 2. Install the first binary

```bash
install -o hestia -g hestia -m 0755 ./hestia-v1.0.0-linux-amd64 /opt/hestia/hestia
```

### 3. Environment file

`/etc/hestia/hestia.env`:

```ini
# --- identity / network -------------------------------------------------
SESSION_SECRET=<long-random-string>          # required
PORT=8070
APP_URL=https://hestia.example.com
APP_VERSION=v1.0.0                           # or bake it via -ldflags

# --- persistence --------------------------------------------------------
DB_PATH=/var/lib/hestia/hestia.db
LOG_PATH=/var/lib/hestia/server.log

# --- first boot only ----------------------------------------------------
# FORCE_BOOTSTRAPPED=true                    # seeds the admin user; remove after first start

# --- self-update: private GitHub Releases --------------------------------
UPDATE_ENABLED=true
UPDATE_GITHUB_OWNER=<owner>
UPDATE_GITHUB_REPO=<private-repo>
UPDATE_GITHUB_ASSET_PATTERN=hestia-{version}-{os}-{arch}
UPDATE_GITHUB_TOKEN=<fine-grained PAT: Contents:Read>

# --- self-update behavior ------------------------------------------------
UPDATE_SYSTEMD=true                          # systemd-native apply (swap + clean exit)
UPDATE_AUTO_APPLY=true                       # apply automatically on the scheduled check
UPDATE_CHECK_SCHEDULE=0 3 * * *              # cron; default @every 24h
UPDATE_DATA_DIR=/var/lib/hestia
# UPDATE_EXECUTABLE_PATH=/opt/hestia/hestia  # only needed if ExecStart is a wrapper script
```

Notes:

- `APP_VERSION` must be the semver of the installed binary. If the binary bakes
  `main.version` via `-ldflags "-X main.version=v1.0.0"`, you can drop the env var.
- `UPDATE_ENABLED=true` with no `UPDATE_GITHUB_*` provider is a startup error —
  that's intentional (fail loudly rather than silently never update).
- `UPDATE_FORWARD_ARGUMENTS` is irrelevant in systemd mode: systemd owns the
  command line (`ExecStart`), not the app.
- `UPDATE_CHECK_SCHEDULE` accepts standard cron or `@every 24h`-style syntax.
- The check stages the binary and — with `UPDATE_AUTO_APPLY=true` — applies it.
  With auto-apply off, admins apply explicitly via the API (see Verification).

### 4. systemd unit

`/etc/systemd/system/hestia.service`:

```ini
[Unit]
Description=Hestia self-updating daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=hestia
Group=hestia
EnvironmentFile=/etc/hestia/hestia.env
ExecStart=/opt/hestia/hestia
Restart=always
RestartSec=2s
TimeoutStartSec=60s

# hardening that is safe with the in-place swap
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictSUIDSGID=true

# NOT allowed: ProtectSystem=strict, ReadOnlyPaths, ProtectSystem=full
# on /opt or /var/lib/hestia — the service user must write the executable and
# the data directory for the swap and the SQLite DB.
```

```bash
systemctl daemon-reload
systemctl enable --now hestia
```

### 5. First boot

With `FORCE_BOOTSTRAPPED=true` set once, the service seeds an admin user. After
it starts, remove that line from `hestia.env` and `systemctl restart hestia`.

### 6. What happens on an update (Section A)

1. The scheduled check (or an admin `check`) downloads the asset, verifies it is
   semver-newer, stages it to `/var/lib/hestia/update`, and records the pending
   version.
2. With auto-apply, or when an admin calls `apply`, the process copies the staged
   binary over `/opt/hestia/hestia` (temp file + atomic rename — safe while the
   process runs), returns an ack, then exits cleanly after ~300 ms.
3. systemd sees the clean exit and `Restart=always` launches `/opt/hestia/hestia`
   — now the new binary — as a **tracked** unit process.
4. On boot, the new process cleans the leftover staged binary and clears the
   stale pending row.

No service state gets lost: the process keeps its DB; only the executable file
changes.

### 7. Verification

```bash
systemctl status hestia                # stays active across updates

# get an admin session
curl -s -X POST http://localhost:8070/api/system/session \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@...","password":"..."}'

# current + staged version
curl -s http://localhost:8070/api/system/updates/status/get
# staged release notes
curl -s http://localhost:8070/api/system/updates/changelog/get
# force a check now
curl -s -X POST http://localhost:8070/api/system/updates/check/create
# explicit apply (when UPDATE_AUTO_APPLY is off)
curl -s -X POST http://localhost:8070/api/system/updates/update/apply
```

Admin users receive an `update_available` in-app notification (plus email when
an SMTP mailer is configured) when a new release is staged.

---

## Section B — unmodified spawn-and-swap pipeline

Use this only when you must run the original handoff (no `UPDATE_SYSTEMD=true`).
Here `apply` spawns the staged binary as a child and exits; the child replaces
the executable and keeps serving. The unit must **not** auto-restart:

```ini
[Service]
Type=simple
User=hestia
Group=hestia
EnvironmentFile=/etc/hestia/hestia.env
ExecStart=/opt/hestia/hestia
Restart=no
```

Known consequences (why Section A exists):

- **Unit state:** after each update the old process exits cleanly, so the unit
  reports `inactive` even though the reparented child is serving fine.
- **`systemctl stop` won't stop it:** the running process is not the tracked
  unit process. Stop it with the process tree: `pkill -f /opt/hestia/hestia`.
- **No crash recovery:** a crash after the handoff is invisible to systemd.
  Add an external watchdog (below).
- **`Restart=always` is broken here:** systemd would spawn a *second* instance
  that races the reparented child for the SQLite DB lock and port, then hit the
  start-limit burst and land the unit in `failed` while the app actually runs.

### Watchdog (optional, for Section B)

A timer that restarts the unit only when nothing is listening on the port:

`/etc/systemd/system/hestia-watchdog.service`:

```ini
[Service]
Type=oneshot
ExecStart=/usr/local/bin/hestia-watchdog
```

`/etc/systemd/system/hestia-watchdog.timer`:

```ini
[Timer]
OnCalendar=minutely
Persistent=true

[Install]
WantedBy=timers.target
```

`/usr/local/bin/hestia-watchdog`:

```bash
#!/bin/bash
PORT=8070
if ! ss -ltn 2>/dev/null | grep -q ":${PORT} "; then
  systemctl restart hestia   # binary at /opt/hestia/hestia is already the new version
fi
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `status` shows `staged_version` empty after a check | No newer release, or the check failed | Look at the app log; confirm a newer tag exists |
| Check errors with a GitHub API `404` | PAT lacks access or repo is wrong | Fine-grained token must have Contents:Read on that exact repo |
| "no asset found matching pattern" | Asset name doesn't exactly match `{version}-{os}-{arch}` | Rename the asset or fix `UPDATE_GITHUB_ASSET_PATTERN`; remember `{version}` includes the `v` |
| Update never triggers | `APP_VERSION` or the tag isn't semver (`dev` etc.) | Bake a real semver into `main.version`; tag releases as `vX.Y.Z` |
| Unit keeps restarting into `failed` | Running the spawn-and-swap pipeline with `Restart=always` | Use `UPDATE_SYSTEMD=true` + Section A, or `Restart=no` + watchdog |
| Apply fails: "replace executable: permission denied" | Service user can't write the executable or its dir | `chown hestia:hestia` on `/opt/hestia/hestia` and `/opt/hestia` |
| Startup fails: "UPDATE_ENABLED=true requires provider env" | Provider env missing | Set `UPDATE_GITHUB_OWNER/REPO/ASSET_PATTERN` |
| Session endpoint 401 | Admin not yet seeded | First-boot with `FORCE_BOOTSTRAPPED=true` |