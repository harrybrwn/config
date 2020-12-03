# Just another config package

This is only here because I got tired of copying it to my projects. Don't get mad **when** the api changes.

## Get Started

Example:

```go
type Config struct {
    Host string  `yaml:"host" default:"localhost"`
    Port int `default:"8080"`
    Database struct {
        Name string `default:"postgres"`
        Port int `default:"5432"`
    } `yaml:"database"`

    Other string `config:"weird-name"`
}

func main() {
    config.SetConfig(&Config{})
    config.SetType("yaml")
    config.AddPath(".") // look for the config file in "."
    config.SetFilename("config.yml") // look for a file named "config.yaml"
    err := config.ReadConfigFile()
    if err == config.NoConfigFile {
        // handle
    }

    fmt.Println(config.GetInt("port")) // 8080
    fmt.Println(config.GetString("database.name")) // postgres

    fmt.Println(config.Get("Other") == config.Get("weird-name")) // true
}
```

## Default Values

When initializing a configuration struct, the package will look for the struct
tag called `default` and set the default value from the tag value. This feature
only supports a limited number of types such as string types and integer types.

By default, this feature will only work when using the global "getter"
functions like `config.Get` or `config.GetInt` and **will not work for the
actual struct** that is passed to `config.SetConfig`. To set the default
values to the raw config struct, you need to call `config.InitDefaults`.


## Flag Binding

If you want to change config values using command line options, you can bind
the current config struct to a flag set.

```go
// test.go
import "flag"

type Config struct {
    Name string `config:"name,shorthand=n,usage=give the name"`
    Inner struct {
        Val int `config:"val,usage=nested flag"`
    } `config:"inner"`
}
config.SetConfig(&Config{})
config.BindToFlagSet(flag.CommandLine)
flag.Parse()
```

Keep in mind that **function call order matters** here. Calling
`config.BindToFlagSet` before `config.SetConfig` means that there is no current
config struct and will most likely result in a segmentation-fault.

```sh
$ go run ./test.go -help
```

```
Usage of /tmp/go-build123456789/b001/exe/test:
  -name value
        give the name
  -inner-val value
        nested flag
```

This feature also supports the common flag package drop in replacement called
`github.com/spf13/pflag` and can be accessed using `BindToPFlagSet(set *pflag.FlagSet)`.
The `shorthand` option is only used with this package.

## TODO

- Add an option to change the nested flag name delimiter. Right now its `-`.
- Add support for multiple config file names.
- Consider using **struct comments** as flag usage if there is no "usage" in the
  struct tag.
