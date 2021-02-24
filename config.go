package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

var (
	// ErrNoConfigFile is returned when the config file cannot be found.
	ErrNoConfigFile = errors.New("no config file")
	// ErrNoConfigDir is returned when all of the possible config paths
	// do not exist.
	ErrNoConfigDir = errors.New("no config directory")
	// ErrFieldNotFound is returned when the field of a struct
	// was not found with reflection
	ErrFieldNotFound = errors.New("could not find struct field")
	// ErrWrongType is returned when the wrong type is used
	ErrWrongType = errors.New("wrong type")

	c      *Config
	nilval = reflect.ValueOf(nil)
)

var (
	nestedFlagDelim rune = '-'
)

func init() { c = &Config{} }

// New creates a new config object from a configuration
// struct.
func New(conf interface{}) *Config {
	cfg := &Config{}
	cfg.SetConfig(conf)
	return cfg
}

// Config holds configuration metadata
type Config struct {
	file string

	files []string
	paths []string

	marshal   func(interface{}) ([]byte, error)
	unmarshal func([]byte, interface{}) error
	tag       string

	// Actual config data
	config interface{}
	elem   reflect.Value
}

// SetConfig will set the config struct
func SetConfig(conf interface{}) error { return c.SetConfig(conf) }

// SetConfig will set the config struct
func (c *Config) SetConfig(conf interface{}) error {
	c.config = conf

	c.elem = reflect.ValueOf(conf)
	if c.elem.Kind() == reflect.Ptr {
		c.elem = c.elem.Elem()
	}
	return nil
}

// InitDefaults will find all the default values and set each
// struct field accordingly.
func InitDefaults() error { return c.InitDefaults() }

// InitDefaults will find all the default values and set each
// struct field accordingly.
func (c *Config) InitDefaults() error { return setDefaults(c.elem) }

// GetConfig will return the the config struct that has been
// set by the user but as an interface type.
func GetConfig() interface{} { return c.GetConfig() }

// GetConfig will return the the config struct that has been
// set by the user but as an interface type.
func (c *Config) GetConfig() interface{} {
	return c.config
}

// AddPath will add a path the the list of possible
// configuration folders
func AddPath(path string) { c.AddPath(path) }

// AddPath will add a path the the list of possible
// configuration folders
func (c *Config) AddPath(path string) {
	c.paths = append(c.paths, os.ExpandEnv(path))
}

// Paths returns the slice of folder paths that
// will be searched when looking for a config file.
func Paths() []string { return c.Paths() }

// Paths returns the slice of folder paths that
// will be searched when looking for a config file.
func (c *Config) Paths() []string { return c.paths }

// AddDefaultDirs sets the config and home directories as possible
// config dir options.
//
// If the config dir is found (see os.UserConfigDir) then
// <config dir>/<name> is added to the list of possible config paths.
// If the home dir is found (see os.UserHomeDir) then
// <home dir>/<name> is added to the list of possible config paths.
func AddDefaultDirs(name string) { c.UseDefaultDirs(name) }

// UseDefaultDirs sets the config and home directories as possible
// config dir options.
//
// If the config dir is found (see os.UserConfigDir) then
// "<config dir>/<name>" is added to the list of possible config paths.
// If the home dir is found (see os.UserHomeDir) then
// "<home dir>/.<name>" is added to the list of possible config paths.
//
// These paths are added the list (its different for windows)
//	$XDG_CONFIG_DIR/<name>
//	$HOME/.<name>
func (c *Config) UseDefaultDirs(name string) {
	configDir, err := os.UserConfigDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(configDir, name))
	}
	home, err := os.UserHomeDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(home, "."+name))
	}
}

// AddConfigDir will add a config dir using the user config dir
// (see os.UserConfigDir) and join it with the name given.
//	$XDG_CONFIG_DIR/<name>
func AddConfigDir(name string) error { return c.AddConfigDir(name) }

// AddConfigDir will add a config dir using the user config dir
// (see os.UserConfigDir) and join it with the name given.
//	$XDG_CONFIG_DIR/<name>
func (c *Config) AddConfigDir(name string) error {
	dir, err := os.UserConfigDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(dir, name))
	}
	return err
}

// AddHomeDir will add a config dir using the user home dir
// (see os.UserHomeDir) and join it with the name given and a "."
//	$HOME/.<name>
func AddHomeDir(name string) error { return c.AddHomeDir(name) }

// AddHomeDir will add a config dir using the user home dir
// (see os.UserHomeDir) and join it with the name given and a "."
//	$HOME/.<name>
func (c *Config) AddHomeDir(name string) error {
	dir, err := os.UserHomeDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(dir, "."+name))
	}
	return err
}

// HomeDir will get the user's home directory
func HomeDir() string {
	sudouser := os.Getenv("SUDO_USER")
	if sudouser != "" {
		return filepath.Join("/home", sudouser)
	}
	home, _ := os.UserHomeDir()
	return home
}

// SetType will set the file type of config being used.
func SetType(ext string) error { return c.SetType(ext) }

// SetType will set the file type of config being used.
func (c *Config) SetType(ext string) error {
	switch ext {
	case "yaml", "yml":
		c.marshal = yaml.Marshal
		c.unmarshal = yaml.Unmarshal
		c.tag = "yaml"
	case "json":
		c.marshal = json.Marshal
		c.unmarshal = json.Unmarshal
		c.tag = "json"
	default:
		return fmt.Errorf("unknown config type %s", ext)
	}
	return nil
}

// ReadConfigFile will read in the config file
func ReadConfigFile() error { return c.ReadConfigFile() }

// ReadConfigFile will read in the config file
func (c *Config) ReadConfigFile() error {
	filename, err := c.findFile()
	if err != nil {
		return err
	}
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return c.unmarshal(raw, c.config)
}

// FileUsed will return the file used for
// configuration. If no existing config directory is
// found then this will return an empty string.
func FileUsed() string { return c.FileUsed() }

// FileUsed will return the file used for
// configuration. If no existing config file is
// found then this will return an empty string.
func (c *Config) FileUsed() string {
	f, _ := c.findFile()
	return f
}

func (c *Config) findFile() (string, error) {
	var file string
	for _, path := range c.paths {
		file = filepath.Join(path, c.file)
		if fileExists(file) {
			return file, nil
		}
	}
	return "", ErrNoConfigFile
}

// DirUsed returns the path of the first existing
// config directory.
func DirUsed() string { return c.DirUsed() }

// DirUsed returns the path of the first existing
// config directory.
// If none of the paths exist, then
// The first non-empty path will be returned.
func (c *Config) DirUsed() string {
	var path string
	for _, path = range c.paths {
		// find the first path that exists
		if exists(path) {
			return path
		}
	}
	// If none of the paths exist, return
	// the first non-empty path.
	//
	// TODO This is weird, should not return anything if no paths exist
	for _, path = range c.paths {
		if path != "" {
			return path
		}
	}
	return ""
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}

func fileExists(p string) bool {
	stat, err := os.Stat(p)
	return !os.IsNotExist(err) && !stat.IsDir()
}

// SetFilename sets the config filename.
//
// Deprecated.
// See AddFile
func SetFilename(name string) { c.SetFilename(name) }

// SetFilename sets the config filename.
//
// Deprecated.
// See AddFile
func (c *Config) SetFilename(name string) {
	c.file = name
	c.AddFile(name)
}

// AddFile adds a file name to a list of possible config files
func AddFile(name string) { c.AddFile(name) }

// AddFile adds a file name to a list of possible config files
func (c *Config) AddFile(name string) { c.files = append(c.files, name) }

// SetNestedFlagDelim changed the character used to seperate
// the names of nested flags.
func SetNestedFlagDelim(delim rune) {
	nestedFlagDelim = delim
}

// BindToFlagSet will bind the config struct to a standard library
// flag set
func BindToFlagSet(set *flag.FlagSet) { c.BindToFlagSet(set) }

// BindToFlagSet will bind the config struct to a standard library
// flag set
func (c *Config) BindToFlagSet(set *flag.FlagSet) {
	bindFlags(c.elem, "", set)
}

func bindFlags(elem reflect.Value, basename string, set *flag.FlagSet) {
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	typ := elem.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	n := typ.NumField()
	for i := 0; i < n; i++ {
		fldtyp := typ.Field(i)
		fldval := elem.Field(i)
		name, _, usage, ok := getFlagInfo(fldtyp)
		if !ok {
			continue
		}

		if basename != "" {
			name = basename + string(nestedFlagDelim) + name
		}
		k := fldtyp.Type.Kind()
		if k == reflect.Struct {
			bindFlags(fldval, name, set)
			continue
		} else if k == reflect.Map {
			continue
		}

		// If BoolVar is not used, flag will require a value to be
		// passed to the flag -boolflag=true. Using BooVar allows
		// the usage to change to -boolflag (without the explicit value).
		if fldtyp.Type.Kind() == reflect.Bool && fldval.CanAddr() {
			deflt := fldtyp.Tag.Get("default")
			set.BoolVar(
				fldval.Addr().Interface().(*bool),
				name, deflt == "true", usage,
			)
		} else {
			set.Var(&flagValue{val: &fldval, fld: &fldtyp}, name, usage)
		}
	}
}

// BindToPFlagSet will bind the config object to a pflag set.
// See https://pkg.go.dev/github.com/spf13/pflag?tab=doc
func BindToPFlagSet(set *pflag.FlagSet) { c.BindToPFlagSet(set) }

// BindToPFlagSet will bind the config object to a pflag set.
// See https://pkg.go.dev/github.com/spf13/pflag?tab=doc
func (c *Config) BindToPFlagSet(set *pflag.FlagSet) {
	bindPFlags(c.elem, "", set)
}

func bindPFlags(elem reflect.Value, basename string, set *pflag.FlagSet) {
	var (
		typ = elem.Type()
		n   = typ.NumField()
	)
	for i := 0; i < n; i++ {
		fldtyp := typ.Field(i)
		fldval := elem.Field(i)

		// TODO support maps or decide not to support maps
		if k := fldval.Kind(); k == reflect.Map {
			continue
		}

		name, shorthand, usage, ok := getFlagInfo(fldtyp)
		if !ok {
			// this field was tagged with "notflag"
			continue
		}

		// handle nested structs
		if fldtyp.Type.Kind() == reflect.Struct {
			// TODO add a struct tag to change this name
			bindPFlags(fldval, name, set)
			continue
		}
		if basename != "" {
			name = basename + string(nestedFlagDelim) + name
		}
		flg := &pflag.Flag{
			Name:      name,
			Shorthand: shorthand,
			Usage:     usage,
			DefValue:  fldtyp.Tag.Get("default"),
			Value:     &flagValue{val: &fldval, fld: &fldtyp},
		}
		if flg.DefValue == "" && fldval.CanInterface() {
			flg.DefValue = fmt.Sprintf("%v", fldval.Interface())
		}
		set.AddFlag(flg)
	}
}

func getFlagInfo(field reflect.StructField) (name, shorthand, usage string, isflag bool) {
	var (
		tag   = field.Tag.Get("config")
		parts = strings.Split(tag, ",")
		i     int
	)
	if len(parts) == 0 {
		return
	}

	name = parts[0]
	for _, p := range parts[1:] {
		if p == "notflag" {
			isflag = false
			return
		}

		i = strings.Index(p, "usage=")
		if i != -1 {
			usage = p[i+6:]
			continue
		}
		if i = strings.Index(p, "shorthand="); i != -1 {
			shorthand = p[i+10 : i+11]
			continue
		} else if i = strings.Index(p, "short="); i != -1 {
			shorthand = p[i+6 : i+7]
			continue
		}
	}
	if name == "" {
		name = field.Name
	}
	isflag = true
	return
}

type flagValue struct {
	val *reflect.Value
	fld *reflect.StructField
}

func (fv *flagValue) String() string {
	if fv.val == nil {
		return ""
	}
	if !fv.val.CanInterface() && fv.val.IsZero() {
		return ""
	}
	return fmt.Sprintf("%v", fv.val.Interface())
}

func (fv *flagValue) Set(s string) error {
	val, err := valueFromString(s, fv.fld, fv.val)
	if err != nil {
		return err
	}
	fv.val.Set(val)
	return nil
}

func (fv *flagValue) Type() string {
	return fv.fld.Type.String()
}

// NewConfigCommand creates a new cobra command for configuration
func NewConfigCommand() *cobra.Command {
	var file, dir, edit bool
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage configuration variables.",
		Long:    `The config command helps manage program configuration variables.`,
		Aliases: []string{"conf"},
		RunE: func(cmd *cobra.Command, args []string) error {
			f := FileUsed()
			if file {
				cmd.Println(f)
				return nil
			}
			if dir {
				d := DirUsed()
				if !exists(d) {
					cmd.Print()
				} else {
					cmd.Println(d)
				}
				return nil
			}
			if edit {
				if f == "" {
					return errors.New("no config file found")
				}
				editor := GetString("editor")
				if editor == "" {
					envEditor := os.Getenv("EDITOR")
					if envEditor == "" {
						return errors.New("no editor set (use $EDITOR or set it in the config)")
					}
					editor = envEditor
				}
				stat, err := os.Stat(f)
				if err != nil {
					return err
				}
				var ex *exec.Cmd
				fstat, ok := stat.Sys().(*syscall.Stat_t)
				// if we are on linux and not part of the file's user
				// or user group, then edit as root
				if ok && (fstat.Uid != uint32(os.Getuid()) && fstat.Gid != uint32(os.Getgid())) {
					cmd.Printf("running \"%s %s\" as root\n", editor, f)
					ex = exec.Command("sudo", editor, f)
				} else {
					ex = exec.Command(editor, f)
				}
				ex.Stdout = cmd.OutOrStdout()
				ex.Stderr = cmd.ErrOrStderr()
				ex.Stdin = cmd.InOrStdin()
				return ex.Run()
			}
			return cmd.Usage()
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use: "get", Short: "Get a config variable",
		Run: func(c *cobra.Command, args []string) {
			for _, arg := range args {
				c.Printf("%+v\n", Get(arg))
			}
		}})
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "edit the config file")
	cmd.Flags().BoolVarP(&file, "file", "f", false, "print the config file path")
	cmd.Flags().BoolVarP(&dir, "dir", "d", false, "print the config directory")
	return cmd
}
