package provider

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// typeOf returns a task type's resource object type and fields.
func typeOf(t *testing.T, key string) (*taskType, tftypes.Object) {
	t.Helper()
	tt, err := lookupTaskType(key)
	if err != nil {
		t.Fatal(err)
	}
	s, err := resourceSchema(context.Background(), tt)
	if err != nil {
		t.Fatal(err)
	}
	return tt, objectType(context.Background(), s)
}

// roundTrip converts a configuration into a task and back into state, with
// the configuration as the prior value.
func roundTrip(t *testing.T, key string, attrs map[string]tftypes.Value) (sdk.Task, tftypes.Value, tftypes.Value) {
	t.Helper()
	tt, typ := typeOf(t, key)
	config := configValue(t, key, attrs)
	values, unknown, err := toValues(tt.Fields(), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown) > 0 {
		t.Fatalf("unknown paths %v in a known configuration", unknown)
	}
	task, err := buildTask(tt, values, tt.Lifecycle().Apply)
	if err != nil {
		t.Fatal(err)
	}
	state, err := fromObserved(tt.Fields(), typ, observedValues(task), config, map[string]tftypes.Value{
		idAttribute: tftypes.NewValue(tftypes.String, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	return task, config, state
}

func TestRoundTripScalarsAndMaps(t *testing.T) {
	task, config, state := roundTrip(t, "dokku_ps_scale", map[string]tftypes.Value{
		"app":   tftypes.NewValue(tftypes.String, "api"),
		"scale": tftypes.NewValue(tftypes.Map{ElementType: tftypes.Number}, map[string]tftypes.Value{"web": tftypes.NewValue(tftypes.Number, big.NewFloat(2))}),
	})
	scale := task.(*sdk.PsScaleTask).Scale
	if scale["web"] != 2 {
		t.Errorf("scale = %v", scale)
	}
	if !state.Equal(config) {
		t.Errorf("state %v differs from config %v", state, config)
	}
}

func TestRoundTripDynamicValues(t *testing.T) {
	values := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"replicas": tftypes.Number,
		"image":    tftypes.Object{AttributeTypes: map[string]tftypes.Type{"tag": tftypes.String}},
		"args":     tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.Bool}},
	}}, map[string]tftypes.Value{
		"replicas": tftypes.NewValue(tftypes.Number, big.NewFloat(3)),
		"image":    tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"tag": tftypes.String}}, map[string]tftypes.Value{"tag": tftypes.NewValue(tftypes.String, "v1")}),
		"args":     tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.Bool}}, []tftypes.Value{tftypes.NewValue(tftypes.String, "-v"), tftypes.NewValue(tftypes.Bool, true)}),
	})
	task, config, state := roundTrip(t, "dokku_scheduler_k3s_chart", map[string]tftypes.Value{
		"chart":  tftypes.NewValue(tftypes.String, "traefik"),
		"values": values,
	})
	got := sdk.TaskValues(task)["values"]
	want := map[string]any{"replicas": int64(3), "image": map[string]any{"tag": "v1"}, "args": []any{"-v", true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("values = %#v, want %#v", got, want)
	}
	if !state.Equal(config) {
		t.Errorf("state %v differs from config %v", state, config)
	}

	// A value the server reports differently is rebuilt from what docket
	// observed, as an object and a tuple.
	tt, typ := typeOf(t, "dokku_scheduler_k3s_chart")
	observed := observedValues(task)
	observed["values"] = map[string]any{"replicas": int64(4)}
	rebuilt, err := fromObserved(tt.Fields(), typ, observed, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	var attrs map[string]tftypes.Value
	if err := rebuilt.As(&attrs); err != nil {
		t.Fatal(err)
	}
	back, err := dynamicToGo(attrs["values"])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, map[string]any{"replicas": int64(4)}) {
		t.Errorf("rebuilt values = %#v", back)
	}
}

// userType is the element type of dokku_http_auth_user's users.
func userType(t *testing.T) (tftypes.Set, tftypes.Object) {
	_, typ := typeOf(t, "dokku_http_auth_user")
	set := typ.AttributeTypes["users"].(tftypes.Set)
	return set, set.ElementType.(tftypes.Object)
}

func user(elem tftypes.Object, username, password string) tftypes.Value {
	attrs := map[string]tftypes.Value{}
	for name, at := range elem.AttributeTypes {
		attrs[name] = tftypes.NewValue(at, nil)
	}
	attrs["username"] = tftypes.NewValue(tftypes.String, username)
	attrs["password"] = tftypes.NewValue(tftypes.String, password)
	return tftypes.NewValue(elem, attrs)
}

func TestNestedWriteOnlyFieldsSurviveARead(t *testing.T) {
	set, elem := userType(t)
	tt, typ := typeOf(t, "dokku_http_auth_user")
	prior := configValue(t, "dokku_http_auth_user", map[string]tftypes.Value{
		"app":   tftypes.NewValue(tftypes.String, "api"),
		"users": tftypes.NewValue(set, []tftypes.Value{user(elem, "alice", "s3cr3t")}),
	})

	// A read returns the user with a hash and no password.
	observed := map[string]any{
		"app":    "api",
		"global": false,
		"users":  []any{map[string]any{"username": "alice", "password": "", "hash": "$apr1$abc"}},
	}
	state, err := fromObserved(tt.Fields(), typ, observed, prior, nil)
	if err != nil {
		t.Fatal(err)
	}
	var attrs map[string]tftypes.Value
	_ = state.As(&attrs)
	var users []tftypes.Value
	_ = attrs["users"].As(&users)
	if len(users) != 1 {
		t.Fatalf("users = %v", users)
	}
	var u map[string]tftypes.Value
	_ = users[0].As(&u)
	var password, hash string
	_ = u["password"].As(&password)
	_ = u["hash"].As(&hash)
	if password != "s3cr3t" {
		t.Errorf("password = %q, want the configured value", password)
	}
	if hash != "$apr1$abc" {
		t.Errorf("hash = %q, want the server's", hash)
	}
	if !attrs["global"].IsNull() {
		t.Errorf("global = %v, want null since the config left it out", attrs["global"])
	}
}

func TestUnknownValuesAreReported(t *testing.T) {
	tt, _ := typeOf(t, "dokku_domains")
	config := configValue(t, "dokku_domains", map[string]tftypes.Value{
		"app":     tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"domains": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, tftypes.UnknownValue)}),
	})
	values, unknown, err := toValues(tt.Fields(), config)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(unknown, []string{"app", "domains"}) {
		t.Errorf("unknown = %v", unknown)
	}
	if len(values) != 0 {
		t.Errorf("values = %v, want none", values)
	}
}

func TestSemanticEqualIgnoresSetOrder(t *testing.T) {
	tt, _ := lookupTaskType("dokku_domains")
	f, _ := tt.Field("domains")
	if !semanticEqual(f, []any{"a", "b"}, []any{"b", "a"}) {
		t.Error("an unordered list in another order compared unequal")
	}
	tb, _ := lookupTaskType("dokku_buildpacks")
	g, _ := tb.Field("buildpacks")
	if semanticEqual(g, []any{"a", "b"}, []any{"b", "a"}) {
		t.Error("an ordered list in another order compared equal")
	}
}
