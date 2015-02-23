# goemon

![](https://raw.githubusercontent.com/mattn/goemon/master/data/goemon.png)

Speed up your development.
When you update js files, the page should be reloaded. When you update go files, application should be recompiled, and run it again. Also the page should be reloaded

# Expected directory structure

```
+---assets
|   +- foo.js
+- web.go
```

# Default configuration

|     pattern      |             behavior            |
|------------------|---------------------------------|
| ./assets/\*.js   | minify js, reload page          |
| ./assets/\*.html | reload page                     |
| ./assets/\*.go   | build, restart app, reload page |

## Usage

```
$ goemon go run web.go
```

## Installation

```
$ go get github.com/mattn/goemon/cmd/goemon
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a mattn)
