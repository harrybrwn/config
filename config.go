package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
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
	// List of full filepaths for possible config files.
	filepaths []string
	// List of names that a config file could be (not a directory)
	filenames []string
	// List of directories in which a config file might be.
	paths []string

	marshal       func(v interface{}) ([]byte, error)
	marshalIndent func(v interface{}, prefix, indent string) ([]byte, error)
	unmarshal     func([]byte, interface{}) error
	tag           string

	// Actual config data
	config interface{}
	elem   reflect.Value

	mu sync.Mutex
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
// configuration folders where a file could be found.
// See AddFile to add a file to the list of possible
// files to be read within a configuration search path.
func AddPath(path string) { c.AddPath(path) }

// AddPath will add a path the the list of possible
// configuration folders where a file could be found.
// See AddFile to add a file to the list of possible
// files to be read within a configuration search path.
func (c *Config) AddPath(path string) {
	p := os.ExpandEnv(path)
	if p != "" {
		c.paths = append(c.paths, p)
	}
}

// AddFile will add a filename to the list of possible config
// filenames. This should be the name of a file without any information
// about the directory or location.
//
// A path and filename could be used and will be treated as relative to
// any of the paths added with AddPath but is behavior which is not
// guaranteed to be supported in the future.
//
// To add a directory to the search path, use AddPath.
func AddFile(name string) { c.AddFile(name) }

// AddFile will add a filename to the list of possible config
// filenames. This should be the name of a file without any information
// about the directory or location.
//
// A path and filename could be used and will be treated as relative to
// any of the paths added with AddPath but is behavior which is not
// guaranteed to be supported in the future.
//
// To add a directory to the search path, use AddPath.
func (c *Config) AddFile(name string) {
	c.filenames = append(c.filenames, name)
}

// AddFilepath will add a full filepath to the list of possible
// config files. This is not the same as adding the path and filename
// separately. This does not add a search path or filename to search
// for and should be regarded as hard coding a configuration filepath.
func AddFilepath(filepath string) { c.AddFilepath(filepath) }

// AddFilepath will add a full filepath to the list of possible
// config files. This is not the same as adding the path and filename
// separately. This does not add a search path or filename to search
// for and should be regarded as hard coding a configuration filepath.
func (c *Config) AddFilepath(filepath string) {
	c.filepaths = append(c.filepaths, filepath)
}

// RemoveFile will remove a filename from the
// list of config files names. Essentially the
// inverse operation of AddFilename.
func RemoveFile(name string) { c.RemoveFile(name) }

// RemoveFile will remove a filename from the
// list of config files names. Essentially the
// inverse operation of AddFilename.
func (c *Config) RemoveFile(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.filenames = remove(c.filenames, name)
}

// RemovePath will remove a path from the list of possible
// config file locations.
func RemovePath(path string) { c.RemovePath(path) }

// RemovePath will remove a path from the list of possible
// config file locations.
func (c *Config) RemovePath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paths = remove(c.paths, path)
}

func remove(s []string, value string) []string {
	removed := make([]string, 0, len(s))
	for _, v := range s {
		if v == value {
			continue
		}
		removed = append(removed, v)
	}
	return removed
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
func (c *Config) UseDefaultDirs(dirname string) {
	configDir, err := os.UserConfigDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(configDir, dirname))
	}
	home, err := os.UserHomeDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(home, "."+dirname))
	}
}

// AddUserConfigDir will add a config dir using the user config dir
// (see os.UserConfigDir) and join it with the name given.
//	$XDG_CONFIG_DIR/<dirname>
func AddUserConfigDir(dirname string) error { return c.AddUserConfigDir(dirname) }

// AddUserConfigDir will add a config dir using the user config dir
// (see os.UserConfigDir) and join it with the name given.
//	$XDG_CONFIG_DIR/<dirname>
func (c *Config) AddUserConfigDir(dirname string) error {
	dir, err := os.UserConfigDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(dir, dirname))
	}
	return err
}

// AddUserHomeDir will add a config dir using the user home dir
// (see os.UserHomeDir) and join it with the name given and a "."
//	$HOME/.<name>
func AddUserHomeDir(name string) error { return c.AddUserHomeDir(name) }

// AddUserHomeDir will add a config dir using the user home dir
// (see os.UserHomeDir) and join it with the name given and a "."
//	$HOME/.<name>
func (c *Config) AddUserHomeDir(name string) error {
	dir, err := homeDir()
	if err == nil {
		c.paths = append(c.paths, filepath.Join(dir, "."+name))
	}
	return err
}

func homeDir() (string, error) {
	if runtime.GOOS != "windows" {
		// This should be fine for linux and darwin but I'm not sure about
		// android, netbsd, freebsd, openbsd, plan9, or solaris. If anyone
		// cares enough to make a pull request about wether or not these OSs
		// use sudo feel free.
		user := os.Getenv("SUDO_USER")
		if user != "" {
			return filepath.Join("/home", user), nil
		}
	}
	dir, err := homedir.Dir()
	if err != nil {
		return dir, err
	}
	return dir, nil
}

// HomeDir will get the user's home directory
func HomeDir() string {
	home, _ := homedir.Dir()
	return home
}

// SetType will set the file type of config being used.
func SetType(ext string) error { return c.SetType(ext) }

// SetType will set the file type of config being used.
func (c *Config) SetType(t string) error {
	switch t {
	case "yaml", "yml":
		c.marshal = yaml.Marshal
		c.marshalIndent = func(
			v interface{},
			prefix, indent string,
		) ([]byte, error) {
			return yaml.Marshal(v)
		}
		c.unmarshal = yaml.Unmarshal
		c.tag = "yaml"
	case "json":
		c.marshal = json.Marshal
		c.marshalIndent = json.MarshalIndent
		c.unmarshal = json.Unmarshal
		c.tag = "json"
	default:
		return fmt.Errorf("unknown config type %s", t)
	}
	return nil
}

// ReadConfig will read all the config files.
//
// If multiple config files are found, then the first
// ones found will have the highest precedence and the
// following config files will not overwrite existing
// values.
func ReadConfig() error { return c.ReadConfig() }

// ReadConfig will read all the config files.
//
// If multiple config files are found, then the first
// ones found will have the highest precedence and the
// following config files will not overwrite existing
// values.
func (c *Config) ReadConfig() error {
	return c.readConfigFiles(0)
}

// ReadConfigNoOverwrite will read all config files but will not overwrite
// fields on the config struct if they are not a zero value.
func ReadConfigNoOverwrite() error { return c.ReadConfigNoOverwrite() }

// ReadConfigNoOverwrite will read all config files but will not overwrite
// fields on the config struct if they are not a zero value.
func (c *Config) ReadConfigNoOverwrite() error {
	// use 1 so that the readConfigFiles will always merge
	// a copy insdead of unmarshaling the config in-place.
	return c.readConfigFiles(1)
}

// Deprecated: Use AddFilepath
func ReadConfigFromFile(filepath string) error { return c.ReadConfigFromFile(filepath) }

// Deprecated: Use AddFilepath
func (c *Config) ReadConfigFromFile(filepath string) error {
	raw, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return c.unmarshal(raw, c.config)
}

// Deprecated: use ReadConfig
func ReadConfigFile() error { return c.ReadConfigFile() }

// Deprecated: use ReadConfig
func (c *Config) ReadConfigFile() error { return c.readConfigFiles(0) }

// readConfigFiles will search through all possible config file locations
// to read and marshal the contents into the user config object. The parameter
// `found` is the number of config files that have been previously found and
// read.
//
// If this number is zero, the first existing config file found will
// be marsheled directly into the user config object, all subsequent files
// will read will not overwrite existing values written by previous config files.
// To prevent overwrites by default, pass a number greater than zero.
func (c *Config) readConfigFiles(found int) error {
	var (
		e     error
		start = found // save this until the end
	)
	c.mu.Lock()
	defer c.mu.Unlock()
	filepaths := existingFiles(c)

	for _, filepath := range filepaths {
		raw, err := ioutil.ReadFile(filepath)
		if err != nil && e == nil {
			e = err
			continue
		}

		found++
		// If the first config file is being read,
		// then we should just unmarshal it directly.
		// Otherwise, secondary config files will be read
		// and only new parameters will be merged
		// into the config object. This prevents overwriting
		// existing values.
		if found == 1 {
			err = c.unmarshal(raw, c.config)
			if err != nil {
				e = err
				continue
			}
		} else {
			cp := reflect.New(c.elem.Type()).Interface()
			err = c.unmarshal(raw, cp)
			if err != nil && e == nil {
				e = err
				continue
			}
			err = merge(c.elem, reflect.ValueOf(cp))
			if err != nil && e == nil {
				e = err
				continue
			}
		}
	}

	if found == start {
		return ErrNoConfigFile
	}
	return e
}

func existingFiles(c *Config) []string {
	l := len(c.filepaths) + len(c.paths) + len(c.filenames)
	res := make([]string, 0, l)
	for _, filepath := range c.filepaths {
		if fileExists(filepath) {
			res = append(res, filepath)
		}
	}
	for _, d := range c.paths {
		for _, f := range c.filenames {
			file := filepath.Join(d, f)
			if fileExists(file) {
				res = append(res, file)
			}
		}
	}
	return res
}

func (c *Config) allPossibleFiles() []string {
	res := make([]string, 0, len(c.filepaths)+len(c.filenames)+len(c.paths))
	res = append(res, c.filepaths...)
	for _, p := range c.paths {
		for _, f := range c.filenames {
			res = append(res, filepath.Join(p, f))
		}
	}
	return res
}

// FilesUsed will return a list of all the configuration files
// that exist within the specified search space. This are the
// same files used when calling ReadConfig.
func FilesUsed() []string { return c.FilesUsed() }

// FilesUsed will return a list of all the configuration files
// that exist within the specified search space. This are the
// same files used when calling ReadConfig.
func (c *Config) FilesUsed() []string {
	return existingFiles(c)
}

// FileUsed will return the file used for
// configuration. If no existing config directory is
// found then this will return an empty string.

// Deprecated: use FilesUsed
func FileUsed() string { return c.FileUsed() }

// FileUsed will return the file used for
// configuration. If no existing config file is
// found then this will return an empty string.

// Deprecated: use FilesUsed
func (c *Config) FileUsed() string {
	f, _ := c.findFile()
	return f
}

func (c *Config) findFile() (string, error) {
	var file string
	for _, path := range c.paths {
		for _, f := range c.filenames {
			file = filepath.Join(path, f)
			if fileExists(file) {
				return file, nil
			}
		}
	}
	return "", ErrNoConfigFile
}

// PathsUsed will return all configuration paths
// where there is an existing configuration file.
func PathsUsed() []string {
	return c.PathsUsed()
}

// PathsUsed will return all configuration paths
// where there is an existing configuration file.
func (c *Config) PathsUsed() []string {
	files := existingFiles(c)
	paths := make([]string, 0, len(files))
	for _, f := range files {
		dir, _ := filepath.Split(f)
		paths = append(paths, dir)
	}
	return paths
}

// DirUsed returns the path of the first existing
// config directory.

// Deprecated: use PathsUsed
func DirUsed() string { return c.DirUsed() }

// DirUsed returns the path of the first existing
// config directory.
// If none of the paths exist, then
// The first non-empty path will be returned.

// Deprecated: use PathsUsed
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

// Deprecated: Use AddFile
func SetFilename(name string) { c.SetFilename(name) }

// Deprecated: Use AddFile
func (c *Config) SetFilename(name string) {
	c.AddFile(name)
}

// SetNestedFlagDelim changed the character used to seperate
// the names of nested flags.
func SetNestedFlagDelim(delim rune) {
	nestedFlagDelim = delim
}

type Flag struct {
	name, usage, shorthand string
}

func (f *Flag) Name() string      { return f.name }
func (f *Flag) Usage() string     { return f.usage }
func (f *Flag) Shorthand() string { return f.shorthand }
func (f *Flag) IsFlag() bool      { return true }

type disabledFlag struct{ name string }

func (f *disabledFlag) IsFlag() bool      { return false }
func (f *disabledFlag) Name() string      { return f.name }
func (f *disabledFlag) Usage() string     { return "" }
func (f *disabledFlag) Shorthand() string { return "" }

func DisableFlag(name string) FlagInfo {
	return &disabledFlag{name}
}

func NewFlagInfo(name, shorthand, usage string) FlagInfo {
	return &Flag{name: name, usage: usage, shorthand: shorthand}
}

type FlagInfo interface {
	Name() string
	Usage() string
	Shorthand() string
	IsFlag() bool

	// TODO(generics) add a Default() function that returns a generic value
}

// BindToFlagSet will bind the config struct to a standard library
// flag set
func BindToFlagSet(set *flag.FlagSet, resolvers ...FlagInfo) { c.BindToFlagSet(set, resolvers...) }

// BindToFlagSet will bind the config struct to a standard library
// flag set
func (c *Config) BindToFlagSet(set *flag.FlagSet, resolvers ...FlagInfo) {
	resmap := make(map[string]FlagInfo)
	for _, r := range resolvers {
		resmap[r.Name()] = r
	}
	bindFlags(c.elem, "", set, resmap)
}

func bindFlags(
	elem reflect.Value,
	basename string,
	set *flag.FlagSet,
	resolvers map[string]FlagInfo,
) {
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
		r, ok := resolvers[name]
		if ok {
			if !r.IsFlag() {
				continue
			}
			usage = r.Usage()
			name = r.Name()
		}

		k := fldtyp.Type.Kind()
		if k == reflect.Struct {
			bindFlags(fldval, name, set, resolvers)
			continue
		} else if k == reflect.Map {
			// TODO maybe support maps
			panic(errors.New("maps not supported for flag binding"))
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
func BindToPFlagSet(set *pflag.FlagSet, resolvers ...FlagInfo) { c.BindToPFlagSet(set, resolvers...) }

// BindToPFlagSet will bind the config object to a pflag set.
// See https://pkg.go.dev/github.com/spf13/pflag?tab=doc
func (c *Config) BindToPFlagSet(set *pflag.FlagSet, resolvers ...FlagInfo) {
	resmap := make(map[string]FlagInfo)
	for _, r := range resolvers {
		resmap[r.Name()] = r
	}
	bindPFlags(c.elem, "", set, resmap)
}

func bindPFlags(elem reflect.Value, basename string, set *pflag.FlagSet, resolvers map[string]FlagInfo) {
	var (
		typ = elem.Type()
		n   = typ.NumField()
	)
	for i := 0; i < n; i++ {
		fldtyp := typ.Field(i)
		fldval := elem.Field(i)

		// TODO maybe support maps

		name, shorthand, usage, ok := getFlagInfo(fldtyp)
		if !ok {
			// this field was tagged with "notflag"
			continue
		}
		if basename != "" {
			name = basename + string(nestedFlagDelim) + name
		}
		r, ok := resolvers[name]
		if ok {
			if !r.IsFlag() {
				continue
			}
			shorthand = r.Shorthand()
			usage = r.Usage()
			name = r.Name()
		}

		// handle nested structs
		if fldtyp.Type.Kind() == reflect.Struct {
			// TODO add a struct tag to change this name
			bindPFlags(fldval, name, set, resolvers)
			continue
		} else if k := fldval.Kind(); k == reflect.Map {
			panic(errors.New("maps not supported for flag binding"))
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
		p = strings.Trim(p, " ")
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
func NewConfigCommand() *cobra.Command { return c.NewConfigCommand() }

func (c *Config) NewConfigCommand() *cobra.Command {
	listpaths := func(prefix ...string) string {
		buf := bytes.Buffer{}
		for _, file := range c.allPossibleFiles() {
			if fileExists(file) {
				buf.WriteString(strings.Join(prefix, ""))
				buf.WriteString(file)
				buf.WriteByte('\n')
			}
		}
		return buf.String()
	}
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage configuration variables.",
		Long:    `The config command helps manage program configuration variables.`,
		Aliases: []string{"conf"},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			if file, err := flags.GetBool("file"); err == nil && file {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", listpaths())
				return nil
			}
			if dir, err := flags.GetBool("dir"); err == nil && dir {
				for _, p := range c.PathsUsed() {
					cmd.Println(p)
				}
				return nil
			}

			f := FileUsed()
			if edit, err := flags.GetBool("edit"); err == nil && edit {
				if f == "" {
					return errors.New("no config file found")
				}
				ex, err := runEditor(f)
				if err != nil {
					return err
				}
				ex.Stdout = cmd.OutOrStdout()
				ex.Stderr = cmd.ErrOrStderr()
				ex.Stdin = cmd.InOrStdin()
				return ex.Run()
			}

			if list, err := flags.GetBool("list-all"); err == nil && list {
				for _, f := range c.allPossibleFiles() {
					cmd.Println(f)
				}
				return nil
			}

			b, err := c.marshalIndent(c.config, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", listpaths("# "))
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b)
			return nil
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use: "get", Short: "Get a config variable",
		Run: func(c *cobra.Command, args []string) {
			for _, arg := range args {
				fmt.Fprintf(c.OutOrStdout(), "%+v\n", Get(arg))
			}
		}})
	return cmd
}

func SetDefaultCommandFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.BoolP("edit", "e", false, "edit the config file")
	flags.BoolP("file", "f", false, "print the config files being used")
	flags.BoolP("dir", "d", false, "print the config directories being used")
	flags.BoolP("list-all", "l", false, "list all possible config files whether they exist or not")
}

func init() {
	cobra.AddTemplateFunc("indent", func(s string) string {
		parts := strings.Split(s, "\n")
		for i := range parts {
			parts[i] = "    " + parts[i]
		}
		return strings.Join(parts, "\n")
	})
}

// This is a template for cobra commands that more
// closely imitates the style of the go command help
// message.
var IndentedCobraHelpTemplate = `Usage:{{if .Runnable}}

	{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
	{{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
	{{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
	{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
	{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:

{{.LocalFlags.FlagUsagesWrapped 100 | trimTrailingWhitespaces | indent}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:

{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces | indent}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:
{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
	{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func findEditor() (string, error) {
	editor := GetString("editor")
	if editor == "" {
		envEditor := os.Getenv("EDITOR")
		if envEditor == "" {
			return "", errors.New("no editor set (use $EDITOR or set it in the config)")
		}
		editor = envEditor
	}
	return editor, nil
}
