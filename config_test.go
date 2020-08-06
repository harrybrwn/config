package config

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func cleanup() {
	defaultConfig = &Config{}
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
		SetStruct(&C{})
		os.Setenv("HOME", os.TempDir())
		os.Setenv("USERPROFILE", os.TempDir())
		os.Setenv("home", os.TempDir())
		UseHomeDir("config_test")
		if defaultConfig.paths[0] != filepath.Join(os.TempDir(), ".config_test") {
			t.Error("home dir not set as a path")
		}
	})

	t.Run("WithConfig", func(t *testing.T) {
		defer cleanup()
		SetStruct(&C{})
		os.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), ".config"))
		os.Setenv("AppData", os.TempDir())
		UseConfigDir("config_test")
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
		if defaultConfig.paths[0] != exp {
			t.Errorf("expected %s; got %s", exp, defaultConfig.paths[0])
		}
		defaultConfig.paths = []string{}
		UseDefaultDirs("config_test")
		if defaultConfig.paths[0] != exp {
			t.Error("home dir not set as a path")
		}
	})
	SetStruct(&C{})
	AddPath("$HOME")
	if defaultConfig.paths[0] != os.TempDir() {
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
	matchFn("gopkg.in/yaml.v2.Unmarshal", defaultConfig.unmarshal)
	matchFn("gopkg.in/yaml.v2.Marshal", defaultConfig.marshal)
	if err = SetType("yaml"); err != nil {
		t.Error(err)
	}
	matchFn("gopkg.in/yaml.v2.Unmarshal", defaultConfig.unmarshal)
	matchFn("gopkg.in/yaml.v2.Marshal", defaultConfig.marshal)
	if err = SetType("json"); err != nil {
		t.Error("err")
	}
	matchFn("encoding/json.Unmarshal", defaultConfig.unmarshal)
	matchFn("encoding/json.Marshal", defaultConfig.marshal)
}

func TestReadConfig_Err(t *testing.T) {
	defer cleanup()
	err := ReadConfigFile()
	if err != ErrNoConfigDir {
		t.Error("should return the 'no config dir' error")
	}
	dir := filepath.Join(
		os.TempDir(), fmt.Sprintf("config_test.%s_%d_%d",
			t.Name(), os.Getpid(), time.Now().UnixNano()))
	AddPath(dir)
	if dirused := DirUsed(); dirused != "" {
		t.Error("expected empty dir because config paths do not exist")
	}
	err = ReadConfigFile()
	if err != ErrNoConfigDir {
		t.Error("should return the 'no config dir' error")
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Error(err)
	}
	if dirused := DirUsed(); dirused != dir {
		t.Errorf("wrong DirUsed: got %s; want %s", dirused, dir)
	}
	defer os.Remove(dir)
	SetFilename("config")
	err = ReadConfigFile()
	if err != ErrNoConfigFile {
		t.Error("exected the 'no config file' error")
	}
}

func TestGet(t *testing.T) {
	defer cleanup()
	type C struct {
		S string `config:"a-string"`
	}
	conf := &C{"this is a test"}
	c := New(conf)
	c.SetFilename("test.txt")
	ires := c.Get("a-string")
	s, ok := ires.(string)
	if !ok {
		t.Error("should have returned a string")
	}
	if s != "this is a test" {
		t.Errorf("expected %s; got %s", conf.S, s)
	}
	// testing the panic in Config.get
	defaultConfig = &Config{}
	defer func() {
		r := recover()
		if r == nil {
			t.Error("should have caught a panic")
		}
		cleanup()
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
	SetStruct(&C{5})
	if HasKey(key) {
		t.Error("config struct should not have this key")
	}
	if Get(key) != nil {
		t.Error("expected a nil value")
	}
	if GetInt(key) > 0 {
		t.Error("invalid key should be an invalid value")
	}
	if GetString(key) != "" {
		t.Error("nonexistant key should give an empty string")
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
}

func TestDefaults(t *testing.T) {
	defer cleanup()
	type C struct {
		A  string  `config:"a" env:"TEST_A"`
		B  int     `config:"b" default:"89"`
		TF bool    `config:"truefalse" default:"true"`
		F  float64 `config:"f" env:"PI"`
		F2 float32 `config:"f2" default:"1.3"`
	}
	conf := &C{}
	SetStruct(conf)
	os.Setenv("TEST_A", "testing-value")
	os.Setenv("PI", strconv.FormatFloat(math.Pi, 'f', 15, 64))

	if GetString("a") != "testing-value" {
		t.Error("environment default gave the wrong value")
	}
	if GetInt("b") != 89 {
		t.Error("`default` tag gave the wrong default value")
	}
	if GetBool("truefalse") == false || GetBool("TF") == false {
		t.Error("wrong default boolean value")
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
		TF bool    `config:"truefalse" default:"true"`
		F  float64 `config:"f" env:"PI"`
		F2 float32 `config:"f2" default:"what am i even doing"`
	}
	conf := &C{}
	SetStruct(conf)
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
}

func TestGetMap(t *testing.T) {
	defer cleanup()
	type C struct {
		M      map[string]string `config:"map"`
		Notmap int               `config:"not-map"`
	}
	SetStruct(&C{M: map[string]string{"one": "1", "two": "2"}})
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
		Inner inner
	}
	c := new(Config)
	obj := &C{
		Ints:  []int{1, 2, 3, 4, 5},
		Inner: inner{[]int64{1, 2, 3, 4, 5}},
	}
	c.SetStruct(obj)
	ints := c.GetIntSlice("ints")
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
	int64s := c.GetInt64Slice("Inner.inner-ints")
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
	SetStruct(conf)
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
