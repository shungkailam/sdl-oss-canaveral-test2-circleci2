#!/bin/bash

# Simple shell script wrapper of psql to connect to our AWS RDS instances

function usage {
  echo "Usage: db.sh <scope> <DB name>"
  echo
  echo "For example:"
  echo "  db.sh dev sherlock_shyan"
  echo "  db.sh prod sherlock_prod"
  echo "  db.sh poc sherlock_slb"
  echo
  exit 1
}

if [ "$#" -ne 2 ]; then
    usage
fi

if [ -z "$SQL_PASSWORD" ]
then
    echo "Please set SQL_PASSWORD env var"
    usage
fi


SCOPE=$1
DB=$2

case $SCOPE in
dev)
  HOST=sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com
  ;;
drdev)
  HOST=sherlock-drdev-pg-cluster.cluster-cjkdoimnohax.us-east-2.rds.amazonaws.com
  ;;
prod)
  HOST=sherlock-pg-prod-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com
  ;;
poc)
  HOST=sherlock-pg-poc-us-west-2a.cn6yw4qpwrhi.us-west-2.rds.amazonaws.com
  ;;
*)
  echo "Unknown scope [$SCOPE], allowed values: dev, prod, poc"
  ;;
esac

[[ ! -z "$HOST" ]] && PGPASSWORD=$SQL_PASSWORD psql -U root -h $HOST $DB
