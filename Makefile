test-clear-cache:
	go clean -testcache

test-http:
	go test ./test/nethttp/... -p=1 -v

test-fasthttp:
	go test ./test/fasthttptest/... -p=1 -v

http-run:
	go run http/run.go

fasthttp-run:
	go run fasthttp/run.go

nc-run:
	time nc localhost 1234

nc-run-sh:
	time ./nethttp/testreq.sh | nc localhost 1234

nc-fasthttp-run-sh:
	time ./fasthttp_readtimeout/testreq.sh | nc localhost 1234