module github.com/hr3lxphr6j/ctfile

go 1.13

require (
	github.com/cenkalti/backoff/v3 v3.1.1
	github.com/dimchansky/utfbom v1.1.0
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/tidwall/gjson v1.6.5
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/ratelimit v0.1.0
)

replace github.com/cenkalti/backoff/v3 => github.com/hr3lxphr6j/backoff/v3 v3.1.1-0.20191203064355-bc5ae9e24fba
