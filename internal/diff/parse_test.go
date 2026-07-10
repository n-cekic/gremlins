package diff

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"

	"github.com/go-gremlins/gremlins/internal/configuration"
)

func TestNewWithCmd(t *testing.T) {
	t.Run("must return zero Diff on empty flag", func(t *testing.T) {
		m := &mock{}

		d, err := NewWithCmd(m.call, "/repo", "")

		if !reflect.DeepEqual(d, Diff{}) || err != nil {
			t.Fatal("incorrect result")
		}
	})

	t.Run("must return error on git rev-parse failure", func(t *testing.T) {
		viper.Set(configuration.UnleashDiffRef, "main")

		m := &mock{
			responses: []mockResponse{
				{err: errors.New("not a git repo")},
			},
		}

		_, err := NewWithCmd(m.call, "/repo", "")
		if err == nil {
			t.Error("must return error")
		}
	})

	t.Run("must return error on git diff failure", func(t *testing.T) {
		viper.Set(configuration.UnleashDiffRef, "test")

		m := &mock{
			responses: []mockResponse{
				{output: []byte("/repo\n")},
				{err: errors.New("test")},
			},
		}

		_, err := NewWithCmd(m.call, "/repo", "")
		if err == nil {
			t.Error("must return error")
		}

		if len(m.calls) != 2 {
			t.Fatal("expected 2 cmd calls")
		}

		expectedArgs := []string{"diff", "--merge-base", "test"}

		if m.calls[1].name != "git" || !reflect.DeepEqual(m.calls[1].args, expectedArgs) {
			t.Log("name", m.calls[1].name)
			t.Log("args", m.calls[1].args)
			t.Error("cmd not called properly")
		}
	})

	t.Run("must return diff error", func(t *testing.T) {
		viper.Set(configuration.UnleashDiffRef, "test")

		m := &mock{
			responses: []mockResponse{
				{output: []byte("/repo\n")},
				{output: []byte(testErrDiff)},
			},
		}

		_, err := NewWithCmd(m.call, "/repo", "")
		if err == nil {
			t.Error("must return error")
		}
	})

	t.Run("must return changes", func(t *testing.T) {
		viper.Set(configuration.UnleashDiffRef, "test")

		m := &mock{
			responses: []mockResponse{
				{output: []byte("/repo\n")},
				{output: []byte(testDiff)},
			},
		}

		expected := Diff{
			changes: map[FileName][]Change{
				"test/test": {{StartLine: 44, EndLine: 44}},
			},
			moduleRel: ".",
		}

		result, err := NewWithCmd(m.call, "/repo", "")

		if err != nil || !reflect.DeepEqual(result, expected) {
			t.Log("err", err)
			t.Log("result", result)
			t.Error("unexpected result")
		}
	})

	t.Run("monorepo: moduleRel computed from git root", func(t *testing.T) {
		viper.Set(configuration.UnleashDiffRef, "main")

		m := &mock{
			responses: []mockResponse{
				{output: []byte("/home/user/repo\n")},
				{output: []byte(testMonorepoDiff)},
			},
		}

		expected := Diff{
			changes: map[FileName][]Change{
				"service-a/main.go": {{StartLine: 44, EndLine: 44}},
			},
			moduleRel: "service-a",
		}

		result, err := NewWithCmd(m.call, "/home/user/repo/service-a", "")

		if err != nil || !reflect.DeepEqual(result, expected) {
			t.Log("err", err)
			t.Log("result", result)
			t.Error("unexpected result")
		}
	})

	t.Run("monorepo: relative moduleRoot is resolved to absolute before computing moduleRel", func(t *testing.T) {
		viper.Set(configuration.UnleashDiffRef, "main")

		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(wd) }()

		repoRoot, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(repoRoot, "service-a"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatal(err)
		}

		m := &mock{
			responses: []mockResponse{
				{output: []byte(repoRoot + "\n")},
				{output: []byte(testMonorepoDiff)},
			},
		}

		expected := Diff{
			changes: map[FileName][]Change{
				"service-a/main.go": {{StartLine: 44, EndLine: 44}},
			},
			moduleRel: "service-a",
		}

		result, err := NewWithCmd(m.call, "service-a", "")

		if err != nil || !reflect.DeepEqual(result, expected) {
			t.Log("err", err)
			t.Log("result", result)
			t.Error("unexpected result")
		}
	})
}

type mockResponse struct {
	output []byte
	err    error
}

type mockCall struct {
	name string
	args []string
}

type mock struct {
	calls     []mockCall
	responses []mockResponse
	idx       int
}

func (m *mock) call(name string, args ...string) execCmd {
	m.calls = append(m.calls, mockCall{name: name, args: args})

	return m
}

func (m *mock) CombinedOutput() ([]byte, error) {
	if m.idx >= len(m.responses) {
		return nil, nil
	}
	resp := m.responses[m.idx]
	m.idx++

	return resp.output, resp.err
}

const (
	testDiff = `
diff --git a/test/test b/test/test
index 54051bc..b92c425 100644
--- a/test/test
+++ b/test/test
@@ -41,6 +41,7 @@ const (
 test = "test"
 test = "test"
 test = "test"
+test = "test"
 test = "test"
 test = "test"
 )
`
	testErrDiff = `
diff --git a/test/test b/test/test
index 54051bc..b92c425 100644
--- a/test/test
+++ b/test/test
@@ -41,7 +41,7 @@ const (
 test = "test"
+test = "test"
 test = "test"
 )
`
	testMonorepoDiff = `
diff --git a/service-a/main.go b/service-a/main.go
index 54051bc..b92c425 100644
--- a/service-a/main.go
+++ b/service-a/main.go
@@ -41,6 +41,7 @@ const (
 test = "test"
 test = "test"
 test = "test"
+test = "test"
 test = "test"
 test = "test"
 )
`
)
