// Command examples writes the provider's examples from docket's task catalog:
// a resource configuration per task type built from its documented examples
// and the tasks each one requires, an import command, and a data source
// configuration. tfplugindocs then renders them into docs/.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/dokku/docket/sdk"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
)

const stateField = "state"

func main() {
	out := "examples"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	catalog, err := sdk.Catalog()
	if err != nil {
		log.Fatal(err)
	}
	schemas := map[string]sdk.TaskSchema{}
	for _, t := range catalog.Tasks {
		schemas[t.Type] = t
	}
	for _, dir := range []string{"resources", "data-sources"} {
		if err := os.RemoveAll(filepath.Join(out, dir)); err != nil {
			log.Fatal(err)
		}
	}
	for _, t := range catalog.Tasks {
		g := &generator{schemas: schemas}
		file, first, err := g.resourceFile(t)
		if err != nil {
			log.Fatalf("%s: %v", t.Type, err)
		}
		if file == nil {
			continue
		}
		dir := filepath.Join(out, "resources", t.Type)
		write(filepath.Join(dir, "resource.tf"), file.Bytes())
		if !readable(t) {
			continue
		}
		task, err := buildTask(t, first.values)
		if err != nil {
			log.Fatalf("%s: %v", t.Type, err)
		}
		address := sdk.IdentityAddress(t.Type, task)
		write(filepath.Join(dir, "import.sh"), []byte(fmt.Sprintf("terraform import %s.%s '%s'\n", t.Type, first.name, address)))
		write(filepath.Join(out, "data-sources", t.Type, "data-source.tf"), dataSourceFile(t, first.values).Bytes())
	}
	write(filepath.Join(out, "data-sources", "dokku_export", "data-source.tf"), []byte(exportExample))
}

const exportExample = `# Export one app as a docket recipe.
data "dokku_export" "api" {
  apps   = ["api"]
  format = "yaml"
}

output "recipe" {
  value = data.dokku_export.api.recipe
}
`

func write(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Fatal(err)
	}
}

// readable reports whether a type can be imported and read as a data source.
func readable(t sdk.TaskSchema) bool {
	return !t.Imperative && t.Probe.Status != sdk.ProbeUnsupported
}

// block is one resource the generator writes.
type block struct {
	typeKey string
	name    string
	values  map[string]any
	// keys are the recipe keys the snippet sets, in the order written.
	keys []string
	// refs maps a recipe key to the traversal that replaces its literal.
	refs      map[string]hcl.Traversal
	dependsOn []string
}

// generator renders one task type's examples into a single file, sharing
// each required resource between the examples that need it.
type generator struct {
	schemas  map[string]sdk.TaskSchema
	blocks   []*block
	byAddr   map[string]*block
	names    map[string]bool
	exampleN int
}

// resourceFile renders the examples of t that the provider can express: one
// whose state is the type's apply state, or present, which declares the same
// members a set does, and the first of those for each resource. It returns
// nil when t has none.
func (g *generator) resourceFile(t sdk.TaskSchema) (*hclwrite.File, *block, error) {
	g.byAddr = map[string]*block{}
	g.names = map[string]bool{}
	var first *block
	comments := map[*block]string{}
	seen := map[string]bool{}
	for _, ex := range t.Examples {
		b, err := g.parse(t.Type, ex.YAML)
		if err != nil {
			return nil, nil, fmt.Errorf("example %q: %w", ex.Name, err)
		}
		state, _ := b.values[stateField].(string)
		if state != "" && state != string(t.Lifecycle.Apply) && state != string(sdk.StatePresent) {
			continue
		}
		if !g.expressible(t, b) {
			continue
		}
		// Two examples of one resource would fight over it in a single
		// configuration, so only the first is kept.
		task, err := buildTask(t, b.values)
		if err != nil {
			return nil, nil, err
		}
		address := sdk.IdentityAddress(t.Type, task)
		if seen[address] {
			continue
		}
		seen[address] = true
		var required []*block
		for _, req := range ex.Requires {
			r, err := g.require(req)
			if err != nil {
				return nil, nil, fmt.Errorf("example %q: %w", ex.Name, err)
			}
			required = append(required, r)
		}
		g.exampleN++
		b.name = "example"
		if g.exampleN > 1 {
			b.name = fmt.Sprintf("example_%d", g.exampleN)
		}
		g.link(b, required)
		g.blocks = append(g.blocks, b)
		comments[b] = ex.Name
		if first == nil {
			first = b
		}
	}
	if first == nil {
		return nil, nil, nil
	}
	file := hclwrite.NewEmptyFile()
	body := file.Body()
	for i, b := range g.blocks {
		if i > 0 {
			body.AppendNewline()
		}
		if c, ok := comments[b]; ok {
			body.AppendUnstructuredTokens(comment(c))
		}
		g.render(body, b)
	}
	return file, first, nil
}

// expressible reports whether a snippet's keys all exist on the resource and
// the task it declares passes docket's checks in the apply state.
func (g *generator) expressible(t sdk.TaskSchema, b *block) bool {
	apply := string(t.Lifecycle.Apply)
	for _, key := range b.keys {
		if key == stateField {
			continue
		}
		f, ok := field(t, key)
		if !ok || slices.Contains(f.ForbiddenWhen, apply) {
			return false
		}
	}
	task, err := buildTask(t, b.values)
	if err != nil {
		return false
	}
	return len(sdk.ConstraintViolations(task)) == 0
}

// require returns the block for a required task, shared by address.
func (g *generator) require(req sdk.ExampleRequirementSchema) (*block, error) {
	b, err := g.parse(req.Type, req.YAML)
	if err != nil {
		return nil, err
	}
	t := g.schemas[req.Type]
	task, err := buildTask(t, b.values)
	if err != nil {
		return nil, err
	}
	address := sdk.IdentityAddress(req.Type, task)
	if existing, ok := g.byAddr[address]; ok {
		return existing, nil
	}
	b.name = g.uniqueName(nameFor(t, b.values), req.Type)
	// A required task depends on what docket derives it needs, among the
	// resources already written.
	var deps []*block
	for _, r := range sdk.TaskRequires(task) {
		if dep, ok := g.byAddr[sdk.IdentityAddress(sdk.TaskTypeOf(r), r)]; ok {
			deps = append(deps, dep)
		}
	}
	g.link(b, deps)
	g.byAddr[address] = b
	g.blocks = append(g.blocks, b)
	return b, nil
}

// link points b at the resources it depends on. A key naming an app or
// service that one of them creates becomes a reference to it, which is what
// orders creation before use. Any other dependency, such as a required
// plugin, is named in depends_on.
func (g *generator) link(b *block, deps []*block) {
	t := g.schemas[b.typeKey]
	b.refs = map[string]hcl.Traversal{}
	referenced := map[*block]bool{}
	for _, obj := range t.Objects {
		if obj.Creates {
			continue
		}
		for _, dep := range deps {
			creator, ok := createdObject(g.schemas[dep.typeKey], obj.Kind)
			if !ok || len(creator.Keys) != len(obj.Keys) {
				continue
			}
			match := true
			for i, key := range obj.Keys {
				if fmt.Sprint(b.values[key]) != fmt.Sprint(dep.values[creator.Keys[i]]) || b.values[key] == nil {
					match = false
				}
			}
			if !match {
				continue
			}
			for i, key := range obj.Keys {
				b.refs[key] = hcl.Traversal{
					hcl.TraverseRoot{Name: dep.typeKey},
					hcl.TraverseAttr{Name: dep.name},
					hcl.TraverseAttr{Name: creator.Keys[i]},
				}
			}
			referenced[dep] = true
			break
		}
	}
	for _, dep := range deps {
		if !referenced[dep] {
			b.dependsOn = append(b.dependsOn, dep.typeKey+"."+dep.name)
		}
	}
}

// createdObject returns the object of kind that a task type creates.
func createdObject(t sdk.TaskSchema, kind sdk.ObjectKind) (sdk.ObjectSchema, bool) {
	for _, obj := range t.Objects {
		if obj.Kind == kind && obj.Creates {
			return obj, true
		}
	}
	return sdk.ObjectSchema{}, false
}

// parse reads a task snippet: its keys in the order written and its values
// typed by the catalog.
func (g *generator) parse(typeKey, snippet string) (*block, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(snippet), &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode || len(doc.Content[0].Content) != 2 {
		return nil, fmt.Errorf("a snippet must hold a single task")
	}
	if got := doc.Content[0].Content[0].Value; got != typeKey {
		return nil, fmt.Errorf("snippet is a %s, not a %s", got, typeKey)
	}
	bodyNode := doc.Content[0].Content[1]
	var keys []string
	for i := 0; i < len(bodyNode.Content); i += 2 {
		keys = append(keys, bodyNode.Content[i].Value)
	}
	body, err := yaml.Marshal(bodyNode)
	if err != nil {
		return nil, err
	}
	task, err := sdk.DecodeTask(typeKey, body)
	if err != nil {
		return nil, err
	}
	all := sdk.TaskValues(task)
	values := map[string]any{}
	for _, key := range keys {
		values[key] = all[key]
	}
	return &block{typeKey: typeKey, keys: keys, values: values}, nil
}

// render writes b into body.
func (g *generator) render(body *hclwrite.Body, b *block) {
	t := g.schemas[b.typeKey]
	res := body.AppendNewBlock("resource", []string{b.typeKey, b.name}).Body()
	for _, key := range b.keys {
		if key == stateField {
			continue
		}
		if ref, ok := b.refs[key]; ok {
			res.SetAttributeTraversal(key, ref)
			continue
		}
		// An empty string, such as the app of a global example, says
		// nothing in HCL.
		if v, ok := b.values[key].(string); ok && v == "" {
			continue
		}
		f, _ := field(t, key)
		res.SetAttributeValue(key, ctyValue(f, b.values[key]))
	}
	if len(b.dependsOn) > 0 {
		var elems []hclwrite.Tokens
		for _, dep := range b.dependsOn {
			parts := strings.Split(dep, ".")
			elems = append(elems, hclwrite.TokensForTraversal(hcl.Traversal{
				hcl.TraverseRoot{Name: parts[0]},
				hcl.TraverseAttr{Name: parts[1]},
			}))
		}
		res.SetAttributeRaw("depends_on", hclwrite.TokensForTuple(elems))
	}
}

// dataSourceFile renders a data source reading the resource an example
// declares, by its identity keys.
func dataSourceFile(t sdk.TaskSchema, values map[string]any) *hclwrite.File {
	file := hclwrite.NewEmptyFile()
	ds := file.Body().AppendNewBlock("data", []string{t.Type, "example"}).Body()
	for _, f := range t.Fields {
		if f.Identity != sdk.IdentityRoleKey {
			continue
		}
		if v, ok := values[f.Name]; ok && !isZero(v) {
			ds.SetAttributeValue(f.Name, ctyValue(f, v))
		}
	}
	return file
}

// ctyValue converts a typed value for writing. A list element's fields that
// hold their zero value are left out, as the snippet left them out.
func ctyValue(f sdk.FieldSchema, value any) cty.Value {
	switch v := value.(type) {
	case nil:
		return cty.NullVal(cty.DynamicPseudoType)
	case string:
		return cty.StringVal(v)
	case bool:
		return cty.BoolVal(v)
	case int64:
		return cty.NumberIntVal(v)
	case float64:
		return cty.NumberFloatVal(v)
	case []any:
		if len(v) == 0 {
			return cty.EmptyTupleVal
		}
		elems := make([]cty.Value, len(v))
		for i, elem := range v {
			if m, ok := elem.(map[string]any); ok && f.Item != nil && f.Item.Type == sdk.TypeObject {
				attrs := map[string]cty.Value{}
				for _, nf := range f.Item.Fields {
					if nv, ok := m[nf.Name]; ok && !isZero(nv) {
						attrs[nf.Name] = ctyValue(nf, nv)
					}
				}
				elems[i] = cty.ObjectVal(attrs)
				continue
			}
			elems[i] = ctyValue(sdk.FieldSchema{}, elem)
		}
		return cty.TupleVal(elems)
	case map[string]any:
		if len(v) == 0 {
			return cty.EmptyObjectVal
		}
		attrs := make(map[string]cty.Value, len(v))
		for key, elem := range v {
			attrs[key] = ctyValue(sdk.FieldSchema{}, elem)
		}
		return cty.ObjectVal(attrs)
	}
	return cty.StringVal(fmt.Sprint(value))
}

func isZero(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case int64:
		return x == 0
	case float64:
		return x == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

// buildTask builds a task from snippet values in the type's apply state.
func buildTask(t sdk.TaskSchema, values map[string]any) (sdk.Task, error) {
	in := map[string]any{}
	for key, value := range values {
		in[key] = value
	}
	in[stateField] = string(t.Lifecycle.Apply)
	return sdk.TaskFromValues(t.Type, in)
}

func field(t sdk.TaskSchema, name string) (sdk.FieldSchema, bool) {
	for _, f := range t.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return sdk.FieldSchema{}, false
}

var nonIdentifier = regexp.MustCompile(`[^a-z0-9_]+`)

// nameFor names a required resource after its identity values, such as
// hello_world for the app hello-world.
func nameFor(t sdk.TaskSchema, values map[string]any) string {
	var parts []string
	for _, key := range t.Identity.Keys {
		if v, ok := values[key]; ok && !isZero(v) {
			if _, isBool := v.(bool); isBool {
				continue
			}
			parts = append(parts, fmt.Sprint(v))
		}
	}
	name := strings.Trim(nonIdentifier.ReplaceAllString(strings.ToLower(strings.Join(parts, "_")), "_"), "_")
	if name == "" || (name[0] >= '0' && name[0] <= '9') {
		name = "r_" + name
	}
	return strings.TrimSuffix(name, "_")
}

// uniqueName returns name, or name followed by the type when another
// resource took it, such as my_app_git_from_image beside the app my_app.
func (g *generator) uniqueName(name, typeKey string) string {
	candidate := name
	if g.names[candidate] {
		candidate = name + "_" + strings.TrimPrefix(typeKey, "dokku_")
	}
	for i := 2; g.names[candidate]; i++ {
		candidate = fmt.Sprintf("%s_%d", name, i)
	}
	g.names[candidate] = true
	return candidate
}

// comment renders a line comment.
func comment(text string) hclwrite.Tokens {
	return hclwrite.Tokens{{Type: hclsyntax.TokenComment, Bytes: []byte("# " + text + "\n")}}
}
