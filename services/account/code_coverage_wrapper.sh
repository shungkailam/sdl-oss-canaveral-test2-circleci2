#!/usr/bin/env sh

# wrapper for code coverage.
# - start code coverage binary as a seperate process
# - upload code coverage results to S3.


echo "Execute Code coverage .."
/usr/src/app/accountserver_cov $@
CODE_COV_PID=$!
echo "Code coverage PID: $CODE_COV_PID"
echo "Code coverage Complete."

echo "Wait for code coverage file to be generated."
sleep 60

echo "Convert code coverage file to html."
cd /tmp/go/src/cloudservices && go tool cover -html=/tmp/accountserver.cov -o /tmp/accountserver_cov.html && cd -

echo "Upload coverage html file to S3."
cd /tmp/go/src/cloudservices && go run cloudmgmt/code_coverage/uploadToS3.go -filePath /tmp/accountserver_cov.html && cd -

echo "Exiting code coverage wrapper script."
