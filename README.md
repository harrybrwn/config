# Just another config package

This is here because [viper](https://github.com/spf13/viper) didn't work for me
on windows.

## Get Started

Example:

```go
type Config struct {
    Host string `yaml:"host" default:"localhost"`
    Port int    `default:"8080"`
    Database struct {
        Name string `default:"postgres"`
        Port int    `default:"5432"`
    } `yaml:"database"`

    Other string `config:"weird-name"`
}

func main() {
    c := &Config{}
    config.SetConfig(c)
    config.SetType("yaml")
    config.AddFile("config.yml") // look for a file named "config.yaml"
    config.AddPath(".")          // look for the config file in "."
    err := config.ReadConfigFile()
    if err == config.NoConfigFile {
        // handle
    }

    // Defaults are set
    fmt.Println(config.GetInt("port")) // 8080
    fmt.Println(config.GetString("database.name")) // postgres
    fmt.Println(config.GetString("database.port")) // 5432

    // Alternate naming
    fmt.Println(config.Get("Other") == config.Get("weird-name")) // true

    // Values are also set on the struct
    fmt.Println(config.GetInt("port") == c.Port) // true
}
```

## Struct tags

For better of for worse, this library relies on struct tags for customization.

| tag     | description                                    |
| ---     | -----------                                    |
| config  | change config name and give other info         |
| default | give the field a default value                 |
| env     | check this environment variable to get a value |


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

### Struct tags for flag binding

| tag       | description                                | example                                     |
| ---       | -----------                                | -------                                     |
| usage     | usage for the flag                         | `config:"name,usage=this is the name flag"` |
| shorthand | give the flag a shorthand (only for pflag) | `config:"name,shorthand=n"`                 |
| notflag   | mark the config field as not a flag        | `config:"file,notflag"`                     |

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
config.BindToFlagSet(
    flag.CommandLine,
    config.NewFlagInfo("name", "n", "give the name"),
    config.NewFlagInfo("inner-val", "", "nested flag"),
)
flag.Parse()
```

Keep in mind that **function call order matters** here. Calling
`config.BindToFlagSet` before `config.SetConfig` means that there is no current
config struct and will most likely result in a nil pointer panic.

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

