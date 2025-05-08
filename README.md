# mapslice

map of append only slices

## build

```
buf lint
buf generate
```

## test
``` 
buf curl -d @ping.json --http2-prior-knowledge http://localhost:8080/jerk.v1.JerkService/Ping
ab -c 100 -n 100000 -p ping.json -T application/json http://localhost:8080/jerk.v1.JerkService/Ping
```