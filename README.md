# saveStatus
Simply receives Transport objects from statusSentry and saves them to CockroachDB database

## local testing 

```shell
PORT=8080 \
DATABASE_URL="XXX"  \
bash localbuild.sh
```