# lime

[![wercker status](https://app.wercker.com/status/5ae8f488a3136a826b480a6bbf33138a/s/master "wercker status")](https://app.wercker.com/project/bykey/5ae8f488a3136a826b480a6bbf33138a)

`lime` is a simple command line utility for live-reloading Go applications.
Just run `lime` in your app directory and your web app will be served with 
`lime` as a proxy. `lime` will automatically recompile your code when it 
detects a change. Your app will be restarted the next time it receives an 
HTTP request.

`lime` adheres to the "silence is golden" principle, so it will only complain 
if there was a compiler error or if you succesfully compile after an error.

## Installation

```shell
go get github.com/kaneshin/lime
```

Make sure that `lime` was installed correctly:

```shell
lime -h
```

## Usage

```shell
cd /path/to/app
lime
```

### Example

```shell
lime -bin=/tmp/bin -ignore-pattern="(\\.git)" -path=./app -immediate=true ./app -version
```

### Options

| option | description |
| :----- | :---------- |
| port             | port for the proxy server |
| app-port         | port for the Go web server |
| bin              | locates built binary |
| ignore-pattern   | pattern to ignore |
| build-pattern    | pattern to build |
| run-pattern      | pattern to run |
| path, -t "."     | watch directory |
| immediate, -i    | run the server immediately after it's built |
| godep, -g        | use godep when building |


## License

[The MIT License (MIT)](http://kaneshin.mit-license.org/)


## Author

Shintaro Kaneko <kaneshin0120@gmail.com>
