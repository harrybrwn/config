package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
)

var pi string

func init()    { pi = strconv.FormatFloat(math.Pi, 'f', 15, 64) }
func cleanup() { c = &Config{} }

func Test(t *testing.T) {
}

func TestPaths(t *testing.T) {
	defer cleanup()
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), ".config"))
	os.Setenv("AppData", os.TempDir())
	os.Setenv("HOME", os.TempDir())
	os.Setenv("USERPROFILE", os.TempDir())
	os.Setenv("home", os.TempDir())

	type C struct{}

	t.Run("WithHome", func(t *testing.T) {
		defer cleanup()
		homedir.DisableCache = true
		SetConfig(&C{})
		os.Setenv("HOME", os.TempDir())
		os.Setenv("USERPROFILE", os.TempDir())
		os.Setenv("home", os.TempDir())
		if err := AddUserHomeDir("config_test"); err != nil {
			t.Error(err)
		}
		exptmp := filepath.Join(os.TempDir(), ".config_test")
		if c.paths[0] != exptmp {
			t.Errorf("home dir not set as a path: got %q, want %q", c.paths[0], exptmp)
		}
		homedir.DisableCache = false
		AddFile("test.txt")
		if c.filenames[0] != "test.txt" {
			t.Errorf("expected %q to be in filenames", "test.txt")
		}
		if c.allPossibleFiles()[0] != filepath.Join(exptmp, "test.txt") {
			t.Error("allPossibleFiles did not return the correct result")
		}
		RemovePath(exptmp)
		RemoveFile("test.txt")
		if len(c.paths) != 0 {
			t.Errorf("should have have any paths: got %v", c.paths)
		}
		if len(c.filenames) != 0 {
			t.Errorf("should have have any filenames: got %v", c.filenames)
		}
	})

	t.Run("WithConfig", func(t *testing.T) {
		defer cleanup()
		SetConfig(&C{})
		os.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), ".config"))
		os.Setenv("AppData", os.TempDir())
		if err := AddUserConfigDir("config_test"); err != nil {
			t.Error(err)
		}
		var exp string
		switch runtime.GOOS {
		case "windows":
			exp = os.TempDir()
		case "darwin":
			exp = filepath.Join(os.TempDir(), "/Library/Application Support")
		case "plan9":
			exp = filepath.Join(os.TempDir(), "lib")
		default:
			exp = filepath.Join(os.TempDir(), ".config")
		}
		exp = filepath.Join(exp, "config_test")
		if c.paths[0] != exp {
			t.Errorf("expected %s; got %s", exp, c.paths[0])
		}
		c.paths = []string{}
		AddDefaultDirs("config_test")
		if c.paths[0] != exp {
			t.Error("home dir not set as a path")
		}
	})
	SetConfig(&C{})
	AddPath("$HOME")
	if c.paths[0] != os.TempDir() {
		t.Error("AddPath did set the wrong path")
	}
	if Paths()[0] != os.TempDir() {
		t.Error("AddPath did set the wrong path")
	}
}

func TestFileTypes(t *testing.T) {
	defer cleanup()
	matchFn := func(n string, i interface{}) {
		name, err := url.QueryUnescape(runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name())
		if err != nil {
			t.Error(err)
		}
		if name != n {
			t.Errorf("wrong function name: want %s; got %s", n, name)
		}
	}
	err := SetType("invalid")
	if err == nil {
		t.Error("expected an invalid filetype error")
	}
	if err = SetType("yml"); err != nil {
		t.Error(err)
	}
	matchFn("gopkg.in/yaml.v2.Unmarshal", c.unmarshal)
	matchFn("gopkg.in/yaml.v2.Marshal", c.marshal)
	if err = SetType("yaml"); err != nil {
		t.Error(err)
	}
	matchFn("gopkg.in/yaml.v2.Unmarshal", c.unmarshal)
	matchFn("gopkg.in/yaml.v2.Marshal", c.marshal)
	if err = SetType("json"); err != nil {
		t.Error("err")
	}
	matchFn("encoding/json.Unmarshal", c.unmarshal)
	matchFn("encoding/json.Marshal", c.marshal)
}

func TestReadConfig_Err(t *testing.T) {
	type C struct {
		Val int `config:"val"`
	}
	defer cleanup()
	check := func(e error) {
		if e != nil {
			t.Error(e)
		}
	}
	var err error
	SetConfig(&C{})
	dir := filepath.Join(
		os.TempDir(), fmt.Sprintf("config_test.%s_%d_%d",
			t.Name(), os.Getpid(), time.Now().UnixNano()))
	AddPath("/tmp/some/path")
	AddPath(dir)
	if dirused := DirUsed(); dirused != "/tmp/some/path" {
		t.Error("DirUsed should be the first non-empty path if none exist")
	}
	if err = ReadConfig(); err != ErrNoConfigFile {
		t.Error("should return the 'no config dir' error")
	}

	check(os.MkdirAll(dir, 0700))
	defer os.RemoveAll(dir)

	if dirused := DirUsed(); dirused != dir {
		t.Errorf("wrong DirUsed: got %s; want %s", dirused, dir)
	} else if dirused == "/tmp/some/path" {
		t.Error("the dummy path should not exist!")
	}
	SetFilename("config")
	check(SetType("yml"))
	err = ReadConfig()
	if err != ErrNoConfigFile {
		t.Errorf("exected the 'no config file' error; got '%v'", err)
	}
	fake := filepath.Join(dir, "config")
	check(os.Mkdir(fake, 0700))
	err = ReadConfig()
	if err == nil {
		t.Error("expected an error while reading the config")
	}
	check(os.Remove(fake))
	check(ioutil.WriteFile(fake, []byte(`val: 10`), 0600))
	check(ReadConfig())
	if FileUsed() != fake {
		t.Error("files should be the same")
	}
}

// Test the ability to load from two different config files
// and not to override the data every time a new one has been
// unmarshalled.
func TestMultiConfigMerging(t *testing.T) {
	defer cleanup()
	type DB struct {
		User string
		Host net.IP
		Port int
	}
	type Email struct {
		Address   string
		Password  string
		Templates []string
	}
	type C struct {
		LogFile string `yaml:"log_file,omitempty"`
		DB      DB
		Email   Email
	}

	var (
		err   error
		dirs  = [2]string{filepath.Join(os.TempDir(), "one"), filepath.Join(os.TempDir(), "two")}
		confs = [2]string{}
	)
	for i, d := range dirs {
		confs[i] = filepath.Join(d, "test.yaml")
		if err = os.MkdirAll(d, 0744); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(d)
	}
	err = ioutil.WriteFile(confs[0], []byte(`
log_file: /tmp/logs/output.log
db:
  user: jimmy
  host: 10.1.1.1
  port: 5432`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(confs[1], []byte(`
log_file: /var/log/testing/out.log
email:
  address: jimmy@my.database.com
  password: insecure
  templates:
    - /usr/local/share/email/template1.txt
    - /usr/local/share/email/template2.txt`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &C{}
	for _, d := range dirs {
		AddPath(d)
	}
	SetConfig(cfg)
	SetType("yaml")
	AddFile("test.yaml")
	err = ReadConfig()
	if err != nil {
		t.Error(err)
	}
	if cfg.LogFile != "/tmp/logs/output.log" {
		t.Errorf("wrong logfile: got %v, want %v", cfg.LogFile, "/tmp/logs/output.log")
	}
	if cfg.DB.User != "jimmy" {
		t.Error("wrong db user")
	}
	if bytes.Compare(cfg.DB.Host, net.ParseIP("10.1.1.1")) != 0 {
		t.Errorf("wong db host: got %v, want %v", cfg.DB.Host, net.ParseIP("10.1.1.1"))
	}
	if cfg.DB.Port != 5432 {
		t.Error("wrong db port")
	}
	if cfg.Email.Address != "jimmy@my.database.com" {
		t.Errorf("bad value: got %v, want %v", cfg.Email.Address, "jimmy@my.database.com")
	}
	if cfg.Email.Password != "insecure" {
		t.Error("wrong password")
	}
	tmpls := []string{"/usr/local/share/email/template1.txt", "/usr/local/share/email/template2.txt"}
	for i, tmpl := range tmpls {
		if cfg.Email.Templates[i] != tmpl {
			t.Errorf("bad template: got %v, want %v", cfg.Email.Templates[i], tmpl)
		}
	}
}

func TestDirUsed(t *testing.T) {
	defer cleanup()
	type C struct {
		S string `config:"s"`
	}
	SetConfig(&C{})
	tmp := os.TempDir()
	AddPath(filepath.Join(tmp, "harrybrwn-config"))
	AddPath(filepath.Join(tmp, "secondary-config"))

	if DirUsed() != filepath.Join(tmp, "harrybrwn-config") {
		t.Error("bad DirUsed result")
	}
	secondary := filepath.Join(tmp, "secondary-config")
	os.Mkdir(secondary, 0777)
	defer os.RemoveAll(secondary)
	if d := DirUsed(); d != secondary {
		t.Errorf("got %q, want %q", d, secondary)
	}
}

func TestGet(t *testing.T) {
	defer cleanup()
	type C struct {
		S string `config:"a-string"`
	}
	conf := C{"this is a test"}
	cfg := New(conf)
	cfg.SetFilename("test.txt")
	ires := cfg.Get("a-string")
	s, ok := ires.(string)
	if !ok {
		t.Error("should have returned a string")
	}
	if s != "this is a test" {
		t.Errorf("expected %s; got %s", conf.S, s)
	}
	if _, err := cfg.GetErr("a-string"); err != nil {
		t.Error(err)
	}
	// testing the panic in Config.get
	c = &Config{}
	defer func() {
		r := recover()
		if r != errElemNotSet {
			t.Error("should have paniced with errElemNotSet")
		}
	}()
	x := Get("a-string")
	if x != nil {
		t.Error("should be nil")
	}
}

func TestGet_Err(t *testing.T) {
	defer cleanup()
	type C struct {
		NotASlice int
	}
	key := "not-here"
	conf := &C{5}
	SetConfig(conf)
	if GetConfig() != conf {
		t.Error("wrong struct pointer")
	}
	if HasKey(key) {
		t.Error("config struct should not have this key")
	}
	if Get(key) != nil {
		t.Error("expected a nil value")
	}
	if _, err := GetErr(key); err == nil {
		t.Error("expected an error")
	}
	if GetInt(key) > 0 {
		t.Error("invalid key should be an invalid value")
	}
	if _, err := GetIntErr(key); err == nil {
		t.Error("expected an error")
	}
	if GetString(key) != "" {
		t.Error("nonexistant key should give an empty string")
	}
	if _, err := GetStringErr(key); err == nil {
		t.Error("expected an error")
	}
	if GetBool(key) {
		t.Error("config struct should not have this key")
	}
	if GetIntSlice(key) != nil {
		t.Error("nonexistant key should give nil int slice")
	}
	if GetInt64Slice(key) != nil {
		t.Error("nonexistant key should give nil int64 slice")
	}
	if GetFloat(key) != 0.0 {
		t.Error("nonexistant key should give a zero value")
	}
	if GetFloat32(key) != 0.0 {
		t.Error("nonexistant key should give a zero value")
	}

	if GetInt("NotASlice") != 5 {
		t.Error("dummy check failed for GetInt")
	}
	if GetIntSlice("NotASlice") != nil {
		t.Error("should return nil for non-slice fields")
	}
	if GetInt64Slice("NotASlice") != nil {
		t.Error("should return nil for non-slice fields")
	}
}

func TestDefaults(t *testing.T) {
	defer cleanup()
	type C struct {
		A  string  `config:"a" env:"TEST_A"`
		B  int     `config:"b" default:"89"`
		TF bool    `config:"truefalse" default:"true"`
		F  float64 `config:"f" env:"PI"`
		F2 float32 `config:"f2" default:"1.3"`
		U  uint    `config:"8"`
	}
	conf := &C{}
	SetConfig(conf)
	os.Setenv("TEST_A", "testing-value")
	os.Setenv("PI", strconv.FormatFloat(math.Pi, 'f', 15, 64))

	if !HasKey("a") {
		t.Error("key 'a' should exist")
	}
	if GetString("a") != "testing-value" {
		t.Error("environment default gave the wrong value")
	}
	if GetInt("b") != 89 {
		t.Error("`default` tag gave the wrong default value")
	}
	if GetBool("truefalse") == false || GetBool("TF") == false {
		t.Error("wrong default boolean value")
	}
	if v, err := GetBoolErr("truefalse"); err != nil || v == false {
		t.Error("wrong value or error:", err)
	}
	if GetFloat("f") != math.Pi {
		t.Error("got wrong float default")
	}
	if GetFloat32("f2") != 1.3 {
		t.Error("wrong defalt float32 value")
	}

	conf.A = "yeet"
	if GetString("a") == "testing-value" {
		t.Error("default string value should have been overridden")
	}
	conf.F = math.E
	if GetFloat("f") == math.Pi {
		t.Error("default float64 value should have been overridden")
	}
	conf.F2 = 5.9
	if GetFloat32("f2") == 1.3 {
		t.Error("default float32 value should have been overridden")
	}
}

func TestDefaults_Err(t *testing.T) {
	defer cleanup()
	type C struct {
		A  string  `config:"a" env:"TEST_A"`
		B  int     `config:"b" default:"x"`
		TF bool    `config:"truefalse" default:"8"`
		F  float64 `config:"f" env:"PI"`
		F2 float32 `config:"f2" default:"what am i even doing"`
	}
	conf := &C{}
	SetConfig(conf)
	os.Setenv("PI", "not a number")

	if _, err := GetIntErr("b"); err == nil {
		t.Error("expected an error")
	}
	if GetInt("b") != 0 {
		t.Error("should not be anything but 0")
	}
	if GetFloat("f") != 0.0 {
		t.Error("default should not be a valid number")
	}
	if GetFloat("f2") != 0 {
		t.Error("default should not be a valid number")
	}
	if _, err := GetBoolErr("truefalse"); err == nil {
		t.Error("expected an error")
	}
}

func TestSetDefaults(t *testing.T) {
	defer cleanup()
	type C struct {
		A         string `config:"a" env:"TEST_A"`
		B         uint   `config:"b" default:"3"`
		NoDefault int8
		TF        bool    `config:"truefalse" default:"true"`
		F         float64 `config:"f" env:"PI"`
		Bytes     []byte  `default:"byte string"`
		Inner     struct {
			Inner struct {
				Inner struct {
					Inner struct {
						Val   string `default:"wow such nesting"`
						Other int    `default:"69"` // shut up, i do what i want
						Thing int16  `default:"99"`
					} `config:"inner"`
				} `config:"inner"`
			} `config:"inner"`
		} `config:"inner"`
	}
	os.Setenv("TEST_A", "testing-auto-defaults")
	os.Setenv("PI", pi)
	conf := &C{}
	SetConfig(conf)
	conf.Inner.Inner.Inner.Inner.Other = 7
	conf.B = 100
	v := reflect.ValueOf(conf).Elem()
	err := setDefaults(v)
	if err != nil {
		t.Error(err)
	}

	if conf.A != "testing-auto-defaults" {
		t.Error("string env default was not set")
	}
	if conf.F != math.Pi {
		t.Error("float env default was not set")
	}
	if conf.TF != true {
		t.Error("bool default was not set")
	}
	if conf.Inner.Inner.Inner.Inner.Thing != 99 {
		t.Error("int16 default was not set")
	}
	if conf.Inner.Inner.Inner.Inner.Val != "wow such nesting" {
		t.Error("string default was not set")
	}
	if conf.B == 3 {
		t.Error("should only set defaults for fields with a zero value")
	}
	if conf.Inner.Inner.Inner.Inner.Other == 69 {
		t.Error("should only set defaults for fields with a zero value")
	}
	if conf.Inner.Inner.Inner.Inner.Other != 7 {
		t.Error("wrong value")
	}
	if GetInt("inner.inner.inner.inner.Other") != 7 {
		t.Error("wrong value")
	}
}

func TestSetDefaults_Err(t *testing.T) {

}

func TestGetMap(t *testing.T) {
	defer cleanup()
	type C struct {
		M      map[string]string `config:"map"`
		Notmap int               `config:"not-map"`
	}
	SetConfig(&C{M: map[string]string{"one": "1", "two": "2"}})
	m := GetStringMap("map")
	if m["one"] != "1" {
		t.Error("wrong map result")
	}
	if m["two"] != "2" {
		t.Error("wrong map result")
	}
	m = GetStringMap("not-map")
	if m != nil {
		t.Error("a non-map should be nil")
	}
	m = GetStringMap("not_here")
	if m != nil {
		t.Error("non-existant key should be nil")
	}
}

func TestSlices(t *testing.T) {
	type inner struct {
		Ints []int64 `config:"inner-ints"`
	}
	type C struct {
		Ints  []int `config:"ints"`
		Inner inner `config:"inner"`
	}
	obj := &C{
		Ints:  []int{1, 2, 3, 4, 5},
		Inner: inner{[]int64{1, 2, 3, 4, 5}},
	}
	SetConfig(obj)
	ints := GetIntSlice("ints")
	expi := 5
	if len(ints) != expi {
		t.Errorf("expected length %d, got length: %d", expi, len(ints))
		return
	}
	for i := range ints {
		if ints[i] != obj.Ints[i] {
			t.Errorf("expected %d; got %d", ints[i], obj.Ints[i])
		}
	}
	if !HasKey("Inner.inner-ints") {
		t.Error("key should exist")
	}
	int64s := GetInt64Slice("Inner.inner-ints")
	if len(int64s) != expi {
		t.Errorf("expected length %d, got length: %d", expi, len(int64s))
		return
	}
	for i := range int64s {
		if int64s[i] != obj.Inner.Ints[i] {
			t.Errorf("expected %d; got %d", ints[i], obj.Inner.Ints[i])
		}
	}
}

func TestSet(t *testing.T) {
	defer cleanup()
	type C struct {
		I int        `config:"i"`
		C complex128 `config:"c"`
	}
	conf := &C{I: 5, C: 5.5i}
	SetConfig(conf)
	if GetInt("i") != 5 {
		t.Error("wrong value")
	}
	if Get("C").(complex128) != 5.5i {
		t.Error("has wrong value")
	}
	if err := set(conf, "i", 10); err != nil {
		t.Error(err)
	}
	if conf.I != 10 {
		t.Error("set did not set the value")
	}
	if err := set(conf, "c", 99.99i); err != nil {
		t.Error(err)
	}
	if conf.C != 99.99i {
		t.Error("did not set the correct value")
	}
}

func TestIsEmpty(t *testing.T) {
	defer cleanup()
	type C struct {
		M      map[string]string `config:"map"`
		Notmap int               `config:"not-map"`
		S      string            `config:"s"`
		I      int               `config:"i"`
	}
	conf := &C{}
	SetConfig(conf)
	if !IsEmpty("s") {
		t.Error("should be empty")
	}
	if !IsEmpty("i") {
		t.Error("should be empty")
	}
	if !IsEmpty("map") {
		t.Error("should be empty")
	}
	conf.I = 1
	conf.S = "hello"
	conf.M = map[string]string{"a": "b"}
	if IsEmpty("s") {
		t.Error("should not be empty")
	}
	if IsEmpty("i") {
		t.Error("should not be empty")
	}
	if IsEmpty("map") {
		t.Error("should not be empty")
	}
}

func TestNestedDelim(t *testing.T) {
	type C struct {
		A struct {
			B int `config:"b"`
		} `config:"a"`
	}
	c := C{}
	SetConfig(&c)
	s := pflag.NewFlagSet("testing", pflag.ContinueOnError)

	BindToPFlagSet(s)
	u := s.FlagUsages()
	if !strings.Contains(u, "a-b") {
		t.Error("wrong flag usage:", u)
	}

	SetNestedFlagDelim('.')
	s = pflag.NewFlagSet("testing", pflag.ContinueOnError)
	BindToPFlagSet(s)
	u = s.FlagUsages()
	if !strings.Contains(u, "a.b") {
		t.Error("wrong flag usage:", u)
	}
}

func TestWatch(t *testing.T) {
	defer cleanup()
	type C struct {
		A string `config:"a" json:"a" default:"hello"`
		B int    `config:"b" json:"b" default:"10"`
	}
	check := func(e error) {
		t.Helper()
		if e != nil {
			t.Error(e)
		}
	}
	conf := &C{B: 12}
	check(SetConfig(conf))
	SetType("json")
	AddPath(os.TempDir())
	AddFile("test.json")
	check(InitDefaults())
	file := filepath.Join(os.TempDir(), "test.json")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(file)
	check(Watch())
	check(ioutil.WriteFile(file, []byte(`{"a":"there"}`), 644))
	time.Sleep(time.Millisecond * 5)

	if conf.A != "there" {
		t.Error("Watch did not update the config struct")
	}
	if conf.B != 12 {
		t.Error("expected 12")
	}
}

func TestUpdated(t *testing.T) {
	defer cleanup()
	type C struct {
		A string `config:"a" json:"a"`
		B int    `config:"b" json:"b"`
	}
	conf := &C{}
	SetConfig(conf)
	SetType("json")
	AddPath(os.TempDir())
	AddFile("test.json")
	file := filepath.Join(os.TempDir(), "test.json")

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(file)

	ch, err := Updated()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err := ioutil.WriteFile(file, []byte(`{"a":"hello","b":12}`), 644)
		if err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Error("update event timeout")
	}
}

func TestEditor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo is not on windows")
	}
	defer cleanup()
	conf := struct{}{}
	SetConfig(&conf)
	os.Setenv("EDITOR", "echo")
	f, err := os.CreateTemp("", "config_editor_test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())
	cmd, err := runEditor(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	cmd.Stderr, cmd.Stdout = &b, &b // make it silent
	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}
}
