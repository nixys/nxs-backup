![nxs-backup](https://github.com/nixys/nxs-backup/assets/28505813/20d0da34-eb6e-4ae4-a5c9-24845407400f)

# nxs-backup

nxs-backup is a tool for creating and delivery backus, rotating it locally and on remote storages, compatible with
GNU/Linux distributions.

## Introduction

### Features

- Full data backup
  - File backups:
    - Discrete files backups
    - Incremental files backups
  - Database backups:
    - Regular backups of MySQL/Mariadb/Percona (5.7/8.0/_all versions_)
    - Xtrabackup (2.4/8.0) of MySQL/Mariadb/Percona (5.7/8.0/all versions)
    - Regular backups of PostgreSQL (9/10/11/12/13/14/15/16/_all versions_)
    - Basebackups of PostgreSQL (9/10/11/12/13/14/15/_all versions_)
    - Backups of MongoDB (3.0/3.2/3.4/3.6/4.0/4.2/4.4/5.0/6.0/7.0/_all versions_)
    - Backups of Redis (_all versions_)
  - Support of user-defined scripts that extend functionality
- Upload and manage backups to the remote storages:
  - S3 (Simple Storage Service that provides object storage through a web interface. Supported by clouds e.g. AWS, GCP)
  - SSH (SFTP)
  - FTP
  - CIFS (SMB)
  - NFS
  - WebDAV
- Fine-tune the database backup process with additional options for optimization purposes
- Notifications about events of the backup process via email and webhooks
- Built-in generator of the configuration files to expedite initial setup
- Easy to read and maintain configuration files with clear transparent structure
- Possibility to restore backups by standard file/database tools (nxs-backup is not required)
- Support of Environment variables in configuration files

### Who can use the tool?

- System Administrators
- DevOps Engineers
- Developers
- Anybody who need to do regular backups

## Quickstart

- Clone the repo
  ```sh
  git clone https://github.com/nixys/nxs-backup.git
  ```
  
### On-premise (bare-metal or virtual machine)

- Go to on-premise directory
  ```sh
  cd nxs-backup/.deploy/on-premise/
  ```
- Install nxs-backup, just download and unpack archive for your CPU architecture.
  ```sh
  curl -L https://github.com/nixys/nxs-backup/releases/latest/download/nxs-backup-amd64.tar.gz -o /tmp/nxs-backup.tar.gz
  tar xf /tmp/nxs-backup.tar.gz -C /tmp
  sudo mv /tmp/nxs-backup /usr/sbin/nxs-backup
  sudo chown root:root /usr/sbin/nxs-backup
  ```
  > [!NOTE]
  > nxs-backup is built for the following processor architectures: amd64 (x86_64), arm (armv7/armv8), arm64 (aarch64).
  > If you need specific version of nxs-backup, or different architecture, you can find it
  on [release page](https://github.com/nixys/nxs-backup/releases).
- Check that installation successful:
  ```sh
  sudo nxs-backup --version
  ```
- Generate configuration files like described [here](docs/USEFUL_INFO.md#generate-configuration-files) or update
  provided `nxs-backup.conf` and jobs configs in `cond.d` dir with your parameters (see [Settings](/docs/settings/README.md) for details)
- For starting nxs-backup process run:
  ```sh
  sudo nxs-backup start
  ```

### Docker-compose

- Go to docker compose directory
  ```sh
  cd nxs-backup/.deploy/docker-compose/
  ```
- Update provided `nxs-backup.conf` file with your parameters (see [Settings](/docs/settings/README.md) for details)
- Launch the nxs-backup with command:
  ```sh
  docker compose up -d --pull
  ```

### Kubernetes

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

## Roadmap

Following features are already in backlog for our development team and will be released soon:

- Encrypting of backups
- Restore backups by nxs-backup
- API for remote management and metrics monitoring
- Web interface for management
- Proprietary startup scheduler
- New backup types (Clickhouse, Elastic, lvm, etc.)
- Programmatic implementation of backup creation instead of calling external utilities
- Ability to set limits on resources utilization
- Update help info

## Feedback

For support and feedback please contact me:

- Telegram: [@r_andreev](https://t.me/r_andreev)
- Email: r.andreev@nixys.io

For news and discussions subscribe the channels:

- Telegram community (news): [@nxs_backup](https://t.me/nxs_backup)
- Telegram community (chat): [@nxs_backup_chat](https://t.me/nxs_backup_chat)

## License

nxs-backup is released under the [GNU GPL-3.0 license](LICENSE).
