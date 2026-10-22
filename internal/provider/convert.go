package provider

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Terraform values cross into docket as maps keyed by recipe key, the form
// sdk.TaskFromValues builds a task from and sdk.TaskValues reads one back as.
// The conversions below work on raw tftypes values, so one generic resource
// handles every task type without per-type models.

// toValues reads the attributes of obj named by fields into a value map. A
// null attribute is left out, so the task takes its default. An unknown one is
// left out too, and its recipe key path is returned so constraint checks can
// treat it as unknown.
func toValues(fields []sdk.FieldSchema, obj tftypes.Value) (map[string]any, []string, error) {
	values := map[string]any{}
	var unknown []string
	if obj.IsNull() {
		return values, nil, nil
	}
	if !obj.IsKnown() {
		for _, f := range fields {
			unknown = append(unknown, f.Name)
		}
		return values, unknown, nil
	}
	var attrs map[string]tftypes.Value
	if err := obj.As(&attrs); err != nil {
		return nil, nil, err
	}
	for _, f := range fields {
		v, ok := attrs[f.Name]
		if !ok {
			continue
		}
		value, paths, err := fieldToGo(f, v, f.Name)
		if err != nil {
			return nil, nil, err
		}
		unknown = append(unknown, paths...)
		if value != nil {
			values[f.Name] = value
		}
	}
	return values, unknown, nil
}

// fieldToGo converts one attribute value, returning nil for a null or unknown
// one and the paths of every unknown value inside it.
func fieldToGo(f sdk.FieldSchema, v tftypes.Value, path string) (any, []string, error) {
	if !v.IsKnown() {
		return nil, []string{path}, nil
	}
	if v.IsNull() {
		return nil, nil, nil
	}
	switch f.Type {
	case sdk.TypeDict:
		if f.Item.Type == sdk.TypeAny {
			if !v.IsFullyKnown() {
				return nil, []string{path}, nil
			}
			value, err := dynamicToGo(v)
			return value, nil, err
		}
		var elems map[string]tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, nil, err
		}
		out := make(map[string]any, len(elems))
		var unknown []string
		for key, elem := range elems {
			elemPath := fmt.Sprintf("%s[%q]", path, key)
			if !elem.IsKnown() {
				unknown = append(unknown, elemPath)
				continue
			}
			value, err := scalarToGo(f.Item.Type, elem)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", elemPath, err)
			}
			out[key] = value
		}
		return out, unknown, nil
	case sdk.TypeList:
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, nil, err
		}
		// docket cannot hold a list element whose value is unknown, so a list
		// with one is unknown as a whole until it is resolved.
		for _, elem := range elems {
			if !elem.IsKnown() {
				return nil, []string{path}, nil
			}
		}
		out := make([]any, 0, len(elems))
		var unknown []string
		for i, elem := range elems {
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			if f.Item.Type == sdk.TypeObject {
				var attrs map[string]tftypes.Value
				if err := elem.As(&attrs); err != nil {
					return nil, nil, err
				}
				item := map[string]any{}
				for _, nf := range f.Item.Fields {
					a, ok := attrs[nf.Name]
					if !ok {
						continue
					}
					value, paths, err := fieldToGo(nf, a, elemPath+"."+nf.Name)
					if err != nil {
						return nil, nil, err
					}
					unknown = append(unknown, paths...)
					if value != nil {
						item[nf.Name] = value
					}
				}
				out = append(out, item)
				continue
			}
			value, err := scalarToGo(f.Item.Type, elem)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", elemPath, err)
			}
			out = append(out, value)
		}
		return out, unknown, nil
	}
	value, err := scalarToGo(f.Type, v)
	return value, nil, err
}

// scalarToGo converts a known, non-null scalar value.
func scalarToGo(t string, v tftypes.Value) (any, error) {
	if v.IsNull() {
		return nil, nil
	}
	switch t {
	case sdk.TypeString:
		var s string
		err := v.As(&s)
		return s, err
	case sdk.TypeBool:
		var b bool
		err := v.As(&b)
		return b, err
	case sdk.TypeInt:
		var n big.Float
		if err := v.As(&n); err != nil {
			return nil, err
		}
		i, accuracy := n.Int64()
		if accuracy != big.Exact {
			return nil, fmt.Errorf("%s is not a whole number in range", n.String())
		}
		return i, nil
	case sdk.TypeFloat:
		var n big.Float
		if err := v.As(&n); err != nil {
			return nil, err
		}
		f, _ := n.Float64()
		return f, nil
	}
	return nil, fmt.Errorf("type %q is not a scalar", t)
}

// dynamicToGo converts a fully known value of any type into the plain Go
// values docket's any fields hold.
func dynamicToGo(v tftypes.Value) (any, error) {
	if v.IsNull() {
		return nil, nil
	}
	typ := v.Type()
	switch {
	case typ.Is(tftypes.String):
		var s string
		err := v.As(&s)
		return s, err
	case typ.Is(tftypes.Bool):
		var b bool
		err := v.As(&b)
		return b, err
	case typ.Is(tftypes.Number):
		var n big.Float
		if err := v.As(&n); err != nil {
			return nil, err
		}
		if i, accuracy := n.Int64(); accuracy == big.Exact {
			return i, nil
		}
		f, _ := n.Float64()
		return f, nil
	case typ.Is(tftypes.List{}), typ.Is(tftypes.Set{}), typ.Is(tftypes.Tuple{}):
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, err
		}
		out := make([]any, len(elems))
		for i, elem := range elems {
			value, err := dynamicToGo(elem)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	case typ.Is(tftypes.Map{}), typ.Is(tftypes.Object{}):
		var elems map[string]tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, err
		}
		out := make(map[string]any, len(elems))
		for key, elem := range elems {
			value, err := dynamicToGo(elem)
			if err != nil {
				return nil, err
			}
			out[key] = value
		}
		return out, nil
	}
	return nil, fmt.Errorf("values of type %s are not supported", typ)
}

// goToTerraform converts a Go value of the shapes sdk.TaskValues returns into
// a value of t. A dynamic type takes the type the value implies: an object for
// a map and a tuple for a list, as HCL writes them.
func goToTerraform(t tftypes.Type, value any) (tftypes.Value, error) {
	if t.Is(tftypes.DynamicPseudoType) {
		dt, err := impliedType(value)
		if err != nil {
			return tftypes.Value{}, err
		}
		return goToTerraform(dt, value)
	}
	if value == nil {
		return tftypes.NewValue(t, nil), nil
	}
	switch {
	case t.Is(tftypes.String):
		s, ok := value.(string)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected a string, got %T", value)
		}
		return tftypes.NewValue(t, s), nil
	case t.Is(tftypes.Bool):
		b, ok := value.(bool)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected a bool, got %T", value)
		}
		return tftypes.NewValue(t, b), nil
	case t.Is(tftypes.Number):
		switch n := value.(type) {
		case int64:
			return tftypes.NewValue(t, new(big.Float).SetInt64(n)), nil
		case int:
			return tftypes.NewValue(t, new(big.Float).SetInt64(int64(n))), nil
		case float64:
			return tftypes.NewValue(t, big.NewFloat(n)), nil
		}
		return tftypes.Value{}, fmt.Errorf("expected a number, got %T", value)
	case t.Is(tftypes.List{}), t.Is(tftypes.Set{}):
		elems, ok := value.([]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected a list, got %T", value)
		}
		var elemType tftypes.Type
		if lt, ok := t.(tftypes.List); ok {
			elemType = lt.ElementType
		} else {
			elemType = t.(tftypes.Set).ElementType
		}
		out := make([]tftypes.Value, len(elems))
		for i, elem := range elems {
			v, err := goToTerraform(elemType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			out[i] = v
		}
		return tftypes.NewValue(t, out), nil
	case t.Is(tftypes.Tuple{}):
		elems, ok := value.([]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected a list, got %T", value)
		}
		tt := t.(tftypes.Tuple)
		out := make([]tftypes.Value, len(elems))
		for i, elem := range elems {
			v, err := goToTerraform(tt.ElementTypes[i], elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			out[i] = v
		}
		return tftypes.NewValue(t, out), nil
	case t.Is(tftypes.Map{}):
		elems, ok := value.(map[string]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected a map, got %T", value)
		}
		elemType := t.(tftypes.Map).ElementType
		out := make(map[string]tftypes.Value, len(elems))
		for key, elem := range elems {
			v, err := goToTerraform(elemType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			out[key] = v
		}
		return tftypes.NewValue(t, out), nil
	case t.Is(tftypes.Object{}):
		elems, ok := value.(map[string]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected an object, got %T", value)
		}
		ot := t.(tftypes.Object)
		out := make(map[string]tftypes.Value, len(ot.AttributeTypes))
		for key, at := range ot.AttributeTypes {
			v, err := goToTerraform(at, elems[key])
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("%s: %w", key, err)
			}
			out[key] = v
		}
		return tftypes.NewValue(t, out), nil
	}
	return tftypes.Value{}, fmt.Errorf("type %s is not supported", t)
}

// impliedType is the Terraform type a dynamic Go value takes.
func impliedType(value any) (tftypes.Type, error) {
	switch v := value.(type) {
	case nil:
		return tftypes.DynamicPseudoType, nil
	case string:
		return tftypes.String, nil
	case bool:
		return tftypes.Bool, nil
	case int, int64, float64:
		return tftypes.Number, nil
	case []any:
		elems := make([]tftypes.Type, len(v))
		for i, elem := range v {
			t, err := impliedType(elem)
			if err != nil {
				return nil, err
			}
			elems[i] = t
		}
		return tftypes.Tuple{ElementTypes: elems}, nil
	case map[string]any:
		attrs := make(map[string]tftypes.Type, len(v))
		for key, elem := range v {
			t, err := impliedType(elem)
			if err != nil {
				return nil, err
			}
			attrs[key] = t
		}
		return tftypes.Object{AttributeTypes: attrs}, nil
	}
	return nil, fmt.Errorf("values of Go type %T are not supported", value)
}

// fromObserved builds the object value of typ for fields from values docket
// observed. prior is the value Terraform already holds for the object - the
// plan after an apply, the state on a refresh, or null - and decides three
// things per attribute:
//
//   - a write-only or apply-only field keeps a non-null prior value, since a
//     read never returns it;
//   - a prior value equal to what docket observed is kept as is, so a set's
//     order, a dynamic value's type and an empty list written as such survive;
//   - a null prior value stays null when docket observed the zero value, so a
//     field left out of the configuration is not read back as "" or false.
//
// Attributes of typ that are not fields, such as id, are taken from extra.
func fromObserved(fields []sdk.FieldSchema, typ tftypes.Object, values map[string]any, prior tftypes.Value, extra map[string]tftypes.Value) (tftypes.Value, error) {
	priorAttrs := map[string]tftypes.Value{}
	if !prior.IsNull() && prior.IsKnown() {
		if err := prior.As(&priorAttrs); err != nil {
			return tftypes.Value{}, err
		}
	}
	out := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for name, at := range typ.AttributeTypes {
		if v, ok := extra[name]; ok {
			out[name] = v
			continue
		}
		out[name] = tftypes.NewValue(at, nil)
	}
	for _, f := range fields {
		at, ok := typ.AttributeTypes[f.Name]
		if !ok {
			continue
		}
		p, hasPrior := priorAttrs[f.Name]
		if !hasPrior {
			p = tftypes.NewValue(at, nil)
		}
		v, err := fieldFromObserved(f, at, values[f.Name], p)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("%s: %w", f.Name, err)
		}
		out[f.Name] = v
	}
	return tftypes.NewValue(typ, out), nil
}

// fieldFromObserved builds one attribute value; see fromObserved.
func fieldFromObserved(f sdk.FieldSchema, t tftypes.Type, observed any, prior tftypes.Value) (tftypes.Value, error) {
	// With no prior value, as on import, such a field takes what the read
	// leaves it at: its default, or null.
	if keptFromConfig(f) {
		if !prior.IsKnown() {
			return tftypes.NewValue(t, nil), nil
		}
		if !prior.IsNull() {
			return prior, nil
		}
	}
	// A value the plan left unknown that the server holds no value for is
	// null, as it would be on import.
	if !prior.IsKnown() && isZero(observed) {
		return tftypes.NewValue(t, nil), nil
	}
	if prior.IsKnown() && prior.IsFullyKnown() {
		if prior.IsNull() {
			if isZero(observed) {
				return prior, nil
			}
		} else if p, _, err := fieldToGo(f, prior, f.Name); err == nil && semanticEqual(f, p, observed) {
			return prior, nil
		}
	}
	if f.Type == sdk.TypeList && f.Item.Type == sdk.TypeObject {
		return objectListFromObserved(f, t, observed, prior)
	}
	if f.Type == sdk.TypeDict && f.Item.Type == sdk.TypeAny && isZero(observed) {
		return tftypes.NewValue(t, nil), nil
	}
	return goToTerraform(t, observed)
}

// objectListFromObserved builds a list or set of objects, matching each
// observed element to the prior element with the same identity keys, so that
// element's write-only fields and null fields carry over.
func objectListFromObserved(f sdk.FieldSchema, t tftypes.Type, observed any, prior tftypes.Value) (tftypes.Value, error) {
	elems, _ := observed.([]any)
	var elemType tftypes.Object
	if lt, ok := t.(tftypes.List); ok {
		elemType = lt.ElementType.(tftypes.Object)
	} else {
		elemType = t.(tftypes.Set).ElementType.(tftypes.Object)
	}
	var priorElems []tftypes.Value
	if prior.IsKnown() && !prior.IsNull() {
		if err := prior.As(&priorElems); err != nil {
			return tftypes.Value{}, err
		}
	}
	out := make([]tftypes.Value, 0, len(elems))
	for _, elem := range elems {
		item, _ := elem.(map[string]any)
		match := tftypes.NewValue(elemType, nil)
		for _, p := range priorElems {
			if !p.IsKnown() || p.IsNull() {
				continue
			}
			var attrs map[string]tftypes.Value
			if err := p.As(&attrs); err != nil {
				return tftypes.Value{}, err
			}
			if sameIdentity(f.Item, attrs, item) {
				match = p
				break
			}
		}
		v, err := fromObserved(f.Item.Fields, elemType, item, match, nil)
		if err != nil {
			return tftypes.Value{}, err
		}
		out = append(out, v)
	}
	return tftypes.NewValue(t, out), nil
}

// sameIdentity reports whether a prior element and an observed one have the
// same identity key values.
func sameIdentity(item *sdk.ItemSchema, prior map[string]tftypes.Value, observed map[string]any) bool {
	if len(item.Keys) == 0 {
		return false
	}
	for _, key := range item.Keys {
		var field sdk.FieldSchema
		for _, nf := range item.Fields {
			if nf.Name == key {
				field = nf
			}
		}
		p, ok := prior[key]
		if !ok || !p.IsKnown() {
			return false
		}
		value, _, err := fieldToGo(field, p, key)
		if err != nil || !reflect.DeepEqual(normalize(value), normalize(observed[key])) {
			return false
		}
	}
	return true
}

// semanticEqual reports whether a prior value and an observed one say the
// same thing: equal ignoring order for an unordered list, ignoring null
// against zero inside list elements, and ignoring write-only and apply-only
// element fields, which a read never fills.
func semanticEqual(f sdk.FieldSchema, prior, observed any) bool {
	if isZero(prior) && isZero(observed) {
		return true
	}
	switch f.Type {
	case sdk.TypeList:
		a, _ := prior.([]any)
		b, _ := observed.([]any)
		if len(a) != len(b) {
			return false
		}
		equal := func(x, y any) bool { return elementEqual(f.Item, x, y) }
		if *f.Ordered {
			for i := range a {
				if !equal(a[i], b[i]) {
					return false
				}
			}
			return true
		}
		used := make([]bool, len(b))
		for _, x := range a {
			found := false
			for j, y := range b {
				if !used[j] && equal(x, y) {
					used[j], found = true, true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(normalize(prior), normalize(observed))
}

// elementEqual compares two list elements.
func elementEqual(item *sdk.ItemSchema, a, b any) bool {
	if item.Type != sdk.TypeObject {
		return reflect.DeepEqual(normalize(a), normalize(b))
	}
	x, _ := a.(map[string]any)
	y, _ := b.(map[string]any)
	for _, nf := range item.Fields {
		if keptFromConfig(nf) {
			continue
		}
		if !semanticEqual(nf, x[nf.Name], y[nf.Name]) {
			return false
		}
	}
	return true
}

// isZero reports whether a value is nil or the zero value of its kind,
// including an empty list or map.
func isZero(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice:
		return rv.Len() == 0
	}
	return rv.IsZero()
}

// normalize makes numbers comparable: every integer becomes an int64.
func normalize(value any) any {
	switch v := value.(type) {
	case int:
		return int64(v)
	case []any:
		out := make([]any, len(v))
		for i, elem := range v {
			out[i] = normalize(elem)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[key] = normalize(elem)
		}
		return out
	}
	return value
}
