![nxs-backup](https://github.com/nixys/go-nxs-backup/assets/28505813/6aa03e3a-db3d-4f34-952b-91cab5fbe49e)

# Quick reference

---

- **Maintained by**:  
  [Nixys LLC](https://nixys.io)

- **Where to get news:**  
  the [@nxs_backup](https://t.me/nxs_backup), [nxs-backup.io](https://nxs-backup.io)

- **Where to get help:**  
  the [@nxs_backup_chat](https://t.me/nxs_backup_chat), [Maintainer Telegram](https://t.me/r_andreev),
  or [GitHub Issues](https://github.com/nixys/nxs-backup/issues)

- **Where to file issues:**  
  https://github.com/nixys/nxs-backup/issues

# Supported tags and respective `Dockerfile` links

---

- [`v3.0.2`, `latest`](https://github.com/nixys/nxs-backup/blob/main/.docker/Dockerfile-alpine)

# What is nxs-backup?

---

nxs-backup is a tool for creating and delivery backus, rotating it locally and on remote storages.

# How to use this image

---

There are two ways described below to install and use the nxs-backup with your infrastructure.

First you need to clone the [repo](https://github.com/nixys/nxs-nackup) and go to `.deploy/docker-compose`
or `.deploy/kubernetes` directory in accordance to the way you choose to install:

```sh
git clone git@github.com:nixys/nxs-backup.git
```

Modify the `nxs-backup.conf` or `helm values` according to your backup issues.

## Docker-compose

Do the following steps:
- Go to docker compose directory
  ```sh
  cd nxs-backup/.deploy/docker-compose/
  ```
- Update provided `nxs-backup.conf` file with your parameters (see [Settings](/docs/settings/README.md) for details)
- Launch the nxs-backup with command:
  ```sh
  docker compose up -d --pull
  ```

## Kubernetes

Do the following steps:

- Go to kubernetes directory
  ```sh
  cd nxs-backup/.deploy/kubernetes/
  ```
- Install [nxs-universal-chart](https://github.com/nixys/nxs-universal-chart) (`Helm 3` is required):
  ```sh
  helm repo add nixys https://registry.nixys.io/chartrepo/public
  ```
- Find examples of `helm values` [here](/docs/example/kubernetes/README.md)
- Fill up your `values.yaml` with correct nxs-backup [Settings](/docs/settings/README.md)
- Launch nxs-backup with command:
  ```sh
  helm -n $NAMESPACE_SERVICE_NAME install nxs-backup nixys/nxs-universal-chart -f values.yaml
  ```
  where $NAMESPACE_SERVICE_NAME is the namespace in which to back up your data

# License

---

[nxs-backup](https://github.com/nixys/nxs-backup) is open source and released under the terms of
the [GNU GPL-3.0 license](https://github.com/nixys/nxs-backup/blob/main/LICENSE).

As with all Docker images, these likely also contain other software which may be under other licenses (such as Bash,
etc. from the base distribution, along with any direct or indirect dependencies of the primary software being
contained).

As for any pre-built image usage, it is the image user's responsibility to ensure that any use of this image complies
with any relevant licenses for all software contained within.
