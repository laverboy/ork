#!/bin/sh

echo "setup start"

aws --endpoint-url=${S3_URL} s3 mb s3://${ENVIRONMENT}-cec-artifact
aws --endpoint-url=${S3_URL} s3api put-bucket-acl --bucket ${ENVIRONMENT}-cec-artifact --acl public-read-write

aws --endpoint-url=${S3_URL} s3 sync /var/empty s3://${ENVIRONMENT}-cec-artifact  --delete

aws --endpoint-url=${S3_URL} s3 cp client-content.json s3://${ENVIRONMENT}-cec-artifact/client-content.json --content-type application/json
echo "setup complete"