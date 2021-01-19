#!/bin/bash
set -e

cd "${PROJECT_ROOT_FOLDER}/deployments"

chmod u+x entrypoint.sh
chmod u+x local.sh
tar -zcvf ${CIRCLE_PROJECT_REPONAME}.tar.gz *
cd ..

mkdir -p ./package/uploads/

mv deployments/${CIRCLE_PROJECT_REPONAME}.tar.gz ./package/uploads/

canaveral/core/scripts/store-build-artifacts.sh
