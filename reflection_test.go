package config

import (
	"flag"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

func TestBindToPFlagSet(t *testing.T) {
	defer cleanup()
	type C struct {
		A          string `config:",usage=this is a test flag,shorthand=a"`
		B          int    `config:"bflag,shorthand=b"`
		OnlyInFile int    `config:",notflag"`
	}

	SetConfig(&C{})
	s := pflag.NewFlagSet("testing", pflag.ContinueOnError)
	s.String("flag", "", "test flag")
	BindToPFlagSet(s)

	s.Parse([]string{"-a", "one", "--bflag=32"})
	if a, err := s.GetString("A"); err != nil {
		t.Error(err)
	} else if a != "one" {
		t.Errorf("expected %q, got %q", "one", a)
	}
	if b, err := s.GetInt("bflag"); err != nil {
		t.Error(err)
	} else if b != 32 {
		t.Error("expected 32, got", b)
	}

	s.Parse([]string{"-b", "69"})
	if b, err := s.GetInt("bflag"); err != nil {
		t.Error(err)
	} else if b != 69 {
		t.Errorf("expected %d, got %d", 69, b)
	}
	if _, err := s.GetInt("OnlyInFile"); err == nil {
		t.Error("expeced an error for a field tagged with \"notflag\"")
	}
}

func TestBindToFlagSet(t *testing.T) {
	defer cleanup()
	type C struct {
		A          string `config:",usage=this is a test flag,shorthand=a"`
		B          int    `config:"bflag,shorthand=b"`
		OnlyInFile int    `config:"test,notflag"`

		Name  string `config:"name,shorthand=n,usage=give the name"`
		Inner struct {
			Val int `config:"val,usage=nested flag"`
		} `config:"inner"`
	}

	SetConfig(&C{})
	s := flag.NewFlagSet("testing", flag.ContinueOnError)
	BindToFlagSet(s)

	if err := s.Parse([]string{"-A", "ONE", "-bflag", "11"}); err != nil {
		t.Error(err)
	}
	f := s.Lookup("A")
	if f.Usage != "this is a test flag" {
		t.Error("go the wrong usage")
	}
	if f.Value.String() != "ONE" {
		t.Errorf("expected %q, got %q", "ONE", f.Value.String())
	}
	f = s.Lookup("bflag")
	if f.Value.String() != "11" {
		t.Error("wrong flag value")
	}
	f = s.Lookup("OnlyInFile")
	if f != nil {
		t.Error("field with notflag tag should not be in set")
	}
	f = s.Lookup("test")
	if f != nil {
		t.Error("field with notflag tag should not be in set")
	}
}

func TestCopyVal(t *testing.T) {
	type Inner struct{ Val string }
	type T struct {
		A    string
		B    int
		C    float32
		Z    string
		Nest *T
		I    Inner
	}
	tt := &T{
		A: "test one", B: 1, C: 1.1, Z: "hello there",
		Nest: &T{A: "this is nested"},
		I:    Inner{Val: "inner value"},
	}

	v := reflect.ValueOf(tt)
	cp := copyVal(v)
	a := v.Elem().Interface().(T)
	b := cp.Interface().(T)

	if a.A != b.A {
		t.Error("value should not change on copy")
	}
	if a.I.Val != b.I.Val {
		t.Error("inner value should have been copied")
	}

	tt.A = "test two"
	tt.B = 2
	tt.C = 2.2
	tt.Z = "general kinobi"

	a = v.Elem().Interface().(T)
	b = cp.Interface().(T)
	if a.A == b.A {
		t.Error("should have changed")
	}
	if a.B == b.B {
		t.Error("should have changed")
	}
	if a.C == b.C {
		t.Error("should have changed")
	}
	if a.Z == b.Z {
		t.Error("should have changed")
	}
	if a.Nest == b.Nest {
		t.Error("pointers should not be the same")
	}
	if b.Nest == nil {
		t.Error("copied nested struct should not be nil")
	}

	ai := 23
	bi := copyVal(reflect.ValueOf(&ai))
	if bi.Interface().(int) != ai {
		t.Error("value should have been copied")
	}
}

func TestCopyVal_MapArr(t *testing.T) {
	type T struct {
		A     string
		B     int
		I     *T
		Slice []string
		Arr   [2]int
	}
	a := map[string]T{
		"one": {A: "hello", B: 1, Slice: []string{"one", "two"}, Arr: [2]int{1, 2}},
		"two": {A: "there", B: 2, I: &T{A: "inner"}},
	}
	b := copyVal(reflect.ValueOf(a)).Interface().(map[string]T)
	if _, ok := b["one"]; !ok {
		t.Fatal("should have the same keys")
	}
	if _, ok := b["two"]; !ok {
		t.Fatal("should have the same keys")
	}
	if b["two"].I == a["two"].I {
		t.Error("should not copy pointer")
	}
	if b["one"].B != a["one"].B {
		t.Error("values should have been copied")
	}
	for i := range a["one"].Slice {
		if a["one"].Slice[i] != b["one"].Slice[i] {
			t.Error("did not copy array")
		}
	}
	for i := range a["one"].Arr {
		if a["one"].Arr[i] != b["one"].Arr[i] {
			t.Error("did not copy array")
		}
	}
}

func TestMerge(t *testing.T) {
	type T struct {
		A    string
		B    int
		C    float32
		Z    string
		Nest *T
		M    map[string]*T
		Arr  []int
	}
	a := &T{A: "one", B: 123, Nest: &T{Z: "string one"}, M: map[string]*T{"key": {A: "in map"}}, Arr: []int{1, 2, 3, 4}}
	b := &T{B: 321, M: map[string]*T{"key": {B: 12}}}

	merge(reflect.ValueOf(b), reflect.ValueOf(a))

	if b.A != a.A {
		t.Error("field A should have been merged")
	}
	if b.B == a.B {
		t.Error("field B should not have been merged")
	}
	if b.Nest == nil {
		t.Error("nested struct pointers should be set")
	}
	if b.Nest == a.Nest {
		t.Error("pointers should not be the same")
	}
	if b.Nest.Z != a.Nest.Z {
		t.Error("all fields on nested struct should be set")
	}
	for i := range a.Arr {
		if a.Arr[i] != b.Arr[i] {
			t.Errorf("merge array: got %v, want %v", b.Arr[i], a.Arr[i])
		}
	}

	if b.M == nil {
		t.Fatal("map was not copied")
	}
	if _, ok := b.M["key"]; !ok {
		t.Fatal("map keys should be copied")
	}
	if b.M["key"] == a.M["key"] {
		t.Fatal("pointers in map should be deep copied")
	}
	if b.M["key"].A != a.M["key"].A {
		t.Error("map value not fully copied")
	}
	if b.M["key"].B != 12 {
		t.Error("lost map data")
	}

	ma := map[string]int{"one": 1, "two": 2}
	mb := map[string]int{"one": 100}
	merge(reflect.ValueOf(&mb), reflect.ValueOf(&ma))

	if mb["one"] != 100 {
		t.Error("existing map value should not be overridden")
	}
	for k := range ma {
		if _, ok := mb[k]; !ok {
			t.Errorf("merged map does not have key %q", k)
		}
	}
	if ma["two"] != mb["two"] {
		t.Error("new values should be set")
	}
}

func TestMerge_Err(t *testing.T) {
	type T struct {
		A    string
		B    int
		C    float32
		Z    string
		Nest *T
		M    map[string]*T
	}
	err := merge(reflect.ValueOf(&T{}), reflect.ValueOf(map[string]string{}))
	if err == nil {
		t.Error("expected an error for different types")
	}
}
