// scriptparam_test.go covers $NAME$ spans and loc engine substitutions.
package game

import "testing"

func TestIsLocEngineValue(t *testing.T) {
	t.Parallel()
	if !IsLocEngineValue("ORDER", "") {
		t.Fatal("$ORDER$ is engine data")
	}
	if !IsLocEngineValue("VALUE", "=+0") {
		t.Fatal("$VALUE|=+0$ is engine data")
	}
	if !IsLocEngineValue("value", "=+0") {
		t.Fatal("$value|=+0$ is engine data")
	}
	if IsLocEngineValue("used_key", "") {
		t.Fatal("$used_key$ is loc reuse")
	}
	if IsLocEngineValue("INDEPENDENCE_WAR_NAME", "") {
		t.Fatal("$INDEPENDENCE_WAR_NAME$ is loc reuse")
	}
	if IsLocEngineValue("used", "U") {
		t.Fatal("$used|U$ is loc reuse")
	}
}
