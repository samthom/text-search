# Searchr

User _docker-compose.yaml_ file to run the dependencies.  
1.  [Redisearch](https://oss.redis.com/redisearch/) - Full text search
2. [Minio](https://min.io/) - S3 Compatible storage platform
3. [Apache Tika](https://tika.apache.org/) - File Parsing

__To run the application__   
1. Run dependencies using `docker-compose up`   
2. Minio dashboard will be available at `localhost:9001`. Login using `ROOT` and `PASSWORD`
3. Create a public bucket as __searchr__ (if exists make it public to access the files publicly)
4. Run the app `make app`
5. Go to `localhost:2112`