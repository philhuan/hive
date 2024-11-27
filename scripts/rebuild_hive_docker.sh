# rebuild if you can't start the hive container
docker rm hive4.0.0
docker run -d -p 10000:10000 -p 10002:10002 --env SERVICE_NAME=hiveserver2 --name hive4.0.0 apache/hive:4.0.0
