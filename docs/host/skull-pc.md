# skull-pc host snapshot (2026-11-09)

Live state of the Ubuntu host that runs the single-node k0s cluster and is also the daily workstation.

## Hardware

| | |
|---|---|
| Board | MSI MS-7C56 (BIOS A.L0, 2026-04-14) |
| CPU | AMD Ryzen 7 5800X, 8C/16T |
| RAM | 31 GB + zram swap 31.3 GB (zstd, ~4:1) |
| GPU | NVIDIA RTX 4070 Ti 12 GB, driver 580.178.04 (`nvidia-driver-580`) |
| NIC | Realtek RTL8111 1 GbE |
| Disk: nvme1n1 (1.8 TB, Crucial P3) | `/boot/efi` (vfat 1G) + `/` ext4 1.8T: Ubuntu, `/var/lib/k0s`, all Longhorn replicas |
| Disk: nvme0n1 (931 GB, Crucial P3) | Windows (NTFS 235G) + `Data` NTFS 695G at `/media/martin/Data` (photos, movies, backups) |

Dual boot with Windows. Windows writes the RTC in local time, which caused the k0s future-dated cert incidents (2026-10-14, 10-22, 10-25, 11-09).

## OS

- Ubuntu 24.04.5 LTS, kernel 7.0.0-38-generic (HWE)
- GNOME on X11 (gdm), PipeWire, IBus
- Timezone Europe/Sofia, RTC in UTC on the Linux side, systemd-timesyncd
- sysctl drop-ins: `99-swappiness.conf` (swappiness 180), `99-zram.conf`, `99-inotify.conf`, `10-map-count.conf`, `50-cursor.conf`
- zram via `systemd-zram-generator` (`zram-size = ram`, zstd, priority 100)

## Kubernetes on the host

- k0s v1.35.3+k0s.0, `k0scontroller.service`:
  `k0s controller --config=/etc/k0s/k0s.yaml --data-dir=/var/lib/k0s --disable-components=konnectivity-server --enable-dynamic-config=true --enable-worker=true --kubelet-extra-args=--max-pods=240 --no-taints=true --profile=custom-eviction`
- Install flags come from `scripts/infra/create-cluster` (k0sctl); re-running it reinstalls the unit and restarts k0s when flags change
- Storage etcd, CNI Calico, kube-proxy iptables
- konnectivity disabled since 2026-11-09 (`--disable-components=konnectivity-server` via k0sctl template, `/var/lib/k0s/manifests/konnectivity` removed)
- Drop-in `/etc/systemd/system/k0scontroller.service.d/`: waits for `/media/martin/Data` mount, `RestartSec=15`, `StartLimitIntervalSec=0`
- Separate `containerd.io` + `docker-ce` 29.8.2 (no containers running), `nvidia-container-toolkit`
- `open-iscsi`/`iscsid` (Longhorn), `nfs-kernel-server` exporting `/media/martin/Data/Backups/home-k8s/Longhorn` to 192.168.1.170 and 10.244.0.0/16
- Drop-in `/etc/systemd/system/nfs-server.service.d/20-data-volume.conf` (mount check + retry forever), `nfs-server-generator` masked

Control-plane footprint (measured 30 min after a restart): ~3.5 GB RAM, ~0.3 core
(kube-apiserver 1.8 GB, etcd 290 MB, containerd + shims ~330 MB, kubelet 210 MB, Calico ~450 MB, rest ~470 MB).

## Host cron jobs

| File | Schedule | Job |
|---|---|---|
| `/etc/cron.d/backup-k0s` | 01:50 daily | `scripts/backup/k0s/backup.sh` (age-encrypted k0s backup to Data) |
| `/etc/cron.d/backup-immich` | 04:00 daily | `scripts/backup/immich/backup.sh` (rsync photos to `/srv/Backups/immich`) |

Metrics for both go to `/var/lib/node-exporter/textfile`.

## System services (running, notable)

`k0scontroller`, `containerd`, `docker`, `nvidia-persistenced`, `iscsid`, `nfs-*` / `rpcbind`, `ssh`, `gnome-remote-desktop`, `smartmontools`, `fwupd`, `cups`, `avahi-daemon`, `ModemManager`, `snapd`, `unattended-upgrades`, `systemd-oomd`, `power-profiles-daemon`

User services: `openclaw-node.service` (OpenClaw node / cua-driver), `a11y-sr-tracer.service` (screen-reader tracer from the Orca investigation).

## Daily-use software

| Kind | Installed |
|---|---|
| Editors / AI | VS Code 1.144 (snap, Claude Code extension), Cursor 3.23 (apt), Codex CLI, OpenClaw CLI (`~/.npm-global`) |
| Browsers | Google Chrome, Firefox (snap), Zen (`/opt/zen`) |
| Chat / media | Slack, Discord, Teams-for-linux, Viber, Thunderbird, Spotify, VLC, GIMP (snaps/apt) |
| Gaming | Steam (snap), Minecraft Launcher, Prism Launcher |
| Devices | DroidCam (`/usr/local/bin/droidcam`) |
| Languages | Node v26.10 / npm 11.19, Python 3.12, Go 1.25.5 (+ gopls, dlv, staticcheck, swag, gotests, impl), gcc/g++/cmake/make |
| Cluster CLIs (`/usr/local/bin`) | kubectl 1.36, helm 4.3, k0s, k0sctl, argocd, flux, vault 1.21, gh 2.88, grpcurl, age, pake |
| `~/.local/bin` | immich-go, agent-007 torrent scripts, Python CLI entry points (uvicorn, fastapi, httpx, opentelemetry-*) |

Desktop memory (same snapshot): GNOME session base ~1.4 GB, Chrome ~2 GB, VS Code ~1.7 GB, Codex ~0.7 GB.

## Leftovers worth cleaning

- `/usr/local/bin/k0s.tmp.*` (2 stale k0s upgrade temp files)
- `nvidia-driver-570` still marked manual next to 580
- `~/.local/bin/agent-007-organize.py.bak-20261008`

## Other machines

- `degradankata` (ssh config, user `it`): unreachable on port 22 at the time of this snapshot, not inventoried.
