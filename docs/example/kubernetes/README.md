# Kubernetes with Helm configuration files example

Here is example of configs for create backups of files and databases for projects deployed in kubernetes.

Files and psql database backup job config at [files+psql.values.yml](files+psql.values.yml) values file
for [nxs-universal-chart](https://github.com/nixys/nxs-universal-chart).

Mysql database backup job config at [mysql.values.yml](mysql.values.yml) values file
for [nxs-universal-chart](https://github.com/nixys/nxs-universal-chart).

Check that cronjob created correct:

* connect to your kubernetes cluster
* get cronjobs list:

  ```sh
  $ kubectl -n $NAMESPACE get cronjobs
  ```
  $NAMESPACE - namespace where you installed nxs-backup
* check that nxs-backup exists in the list of cronjobs