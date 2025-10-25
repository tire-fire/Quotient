#!/bin/sh
set -ex

# Prepare Postgres for replication to a CNPG cluster
echo "host replication all 172.18.0.0/16 scram-sha-256" >> /var/lib/postgresql/data/pg_hba.conf
echo "host replication all 10.42.0.0/16 scram-sha-256" >> /var/lib/postgresql/data/pg_hba.conf
echo "include_if_exists 'custom.conf'" >> /var/lib/postgresql/data/postgresql.conf
