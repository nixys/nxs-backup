# nxs-backup

nxs-backup is a tool for creating and rotating backups locally and on remote storages, compatible with GNU/Linux
distributions.

## Introduction

### Features

* Full data backup
* Discrete and incremental files backups
* Upload and manage backups to the remote storages:
    * S3 (Simple Storage Service that provides object storage through a web interface. Supported by clouds e.g. AWS,
      GCP)
    * ssh (sftp)
    * ftp
    * cifs (smb)
    * nfs
    * WebDAV
* Database backups:
    * Regular backups of MySQL/Mariadb/Percona (5.7/8.0/_all versions_)
    * Xtrabackup (2.4/8.0) of MySQL/Mariadb/Percona (5.7/8.0/all versions)
    * Regular backups of PostgreSQL (9/10/11/12/13/14/15/16/_all versions_)
    * Basebackups of PostgreSQL (9/10/11/12/13/14/15/_all versions_)
    * Backups of MongoDB (3.0/3.2/3.4/3.6/4.0/4.2/4.4/5.0/6.0/7.0/_all versions_)
    * Backups of Redis (_all versions_)
* Fine-tune the database backup process with additional options for optimization purposes
* Notifications via email and webhooks about events in the backup process
* Built-in generator of the configuration files to expedite initial setup
* Easy to read and maintain configuration files with clear transparent structure
* Possibility to restore backups with standard file/database tools (nxs-backup is not required)
* Support of user-defined scripts that extend functionality
* Support of Environment variables in config files

### Who can use the tool?

* System Administrators
* DevOps Engineers
* Developers
* Anybody who need to do regular backups

## Quickstart

### On-premise (bare-metal or virtual machine)

nxs-backup is provided for the following processor architectures: amd64 (x86_64), arm (armv7/armv8), arm64 (aarch64).

To install latest version just download and unpack archive for your CPU architecture.

```bash
curl -L https://github.com/nixys/nxs-backup/releases/latest/download/nxs-backup-amd64.tar.gz -o /tmp/nxs-backup.tar.gz
tar xf /tmp/nxs-backup.tar.gz -C /tmp
sudo mv /tmp/nxs-backup /usr/sbin/nxs-backup
sudo chown root:root /usr/sbin/nxs-backup
```
> [!NOTE]  
> If you need specific version of nxs-backup, or different architecture, you can find it
on [release page](https://github.com/nixys/nxs-backup/releases).

Then check that installation successful:

```sh
sudo nxs-backup --version
```

For starting nxs-backup process run:

```sh
sudo nxs-backup start
```

### Docker-compose

- clone the repo
  ```sh
  git clone https://github.com/nixys/nxs-backup.git
  ```
- go to docker compose directory
  ```sh
  cd nxs-backup/docs/example/docker-compose/
  ```
- update provided `nxs-backup.conf` file with your parameters (see [Settings](docs/settings/README.md) for details)
- launch the nxs-backup with command:
  ```sh
  docker compose up -d --build
  ```

### Kubernetes

Do the following steps:

- install [nxs-universal-chart](https://github.com/nixys/nxs-universal-chart) (`Helm 3` is required):
  ```sh
  helm repo add nixys https://registry.nixys.ru/chartrepo/public
  ```
- launch nxs-backup with command:
  ```sh
  helm -n $NAMESPACE_SERVICE_NAME install nxs-backup nixys/universal-chart -f values.yaml
  ```
  where $NAMESPACE_SERVICE_NAME is namespace with your application launched
- find example `values.yaml` file [here](docs/example/kubernetes+helm/README.md)
- update it according [Settings](docs/settings/README.md)
- configure nxs-backup (see [Configure](docs/example/README.md) for details)

## Roadmap

Following features are already in backlog for our development team and will be released soon:

* Encrypting of backups
* Restore backups by nxs-backup
* API for remote management and metrics monitoring
* Web interface for management
* Proprietary startup scheduler
* New backup types (Clickhouse, Elastic, lvm, etc.)
* Programmatic implementation of backup creation instead of calling external utilities
* Ability to set limits on resource utilization
* Update help info

## Feedback

For support and feedback please contact me:

* telegram: [@r_andreev](https://t.me/r_andreev)
* e-mail: r.andreev@nixys.io

## License

nxs-backup is released under the [GNU GPL-3.0 license](LICENSE).
