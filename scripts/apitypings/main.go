package main

import (
	"context"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"cdr.dev/slog/sloggers/sloghuman"

	"golang.org/x/tools/go/packages"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
)

const (
	baseDir = "./codersdk"
)

// TODO: Handle httpapi.Response and other types
func main() {
	ctx := context.Background()
	log := slog.Make(sloghuman.Sink(os.Stderr))
	codeBlocks, err := GenerateFromDirectory(ctx, log, baseDir)
	if err != nil {
		log.Fatal(ctx, err.Error())
	}

	// Just cat the output to a file to capture it
	fmt.Println(codeBlocks.String())
}

// TypescriptTypes holds all the code blocks created.
type TypescriptTypes struct {
	// Each entry is the type name, and it's typescript code block.
	Types map[string]string
	Enums map[string]string
}

// String just combines all the codeblocks.
func (t TypescriptTypes) String() string {
	var s strings.Builder
	sortedTypes := make([]string, 0, len(t.Types))
	sortedEnums := make([]string, 0, len(t.Types))

	for k := range t.Types {
		sortedTypes = append(sortedTypes, k)
	}
	for k := range t.Enums {
		sortedEnums = append(sortedEnums, k)
	}

	for _, k := range sortedTypes {
		v := t.Types[k]
		s.WriteString(v)
		s.WriteRune('\n')
	}

	for _, k := range sortedEnums {
		v := t.Enums[k]
		s.WriteString(v)
		s.WriteRune('\n')
	}
	return s.String()
}

// GenerateFromDirectory will return all the typescript code blocks for a directory
func GenerateFromDirectory(ctx context.Context, log slog.Logger, directory string) (*TypescriptTypes, error) {
	g := Generator{
		log: log,
	}
	err := g.parsePackage(ctx, directory)
	if err != nil {
		return nil, xerrors.Errorf("parse package %q: %w", directory, err)
	}

	codeBlocks, err := g.generateAll()
	if err != nil {
		return nil, xerrors.Errorf("parse package %q: %w", directory, err)
	}

	return codeBlocks, nil
}

type Generator struct {
	// Package we are scanning.
	pkg *packages.Package
	log slog.Logger
}

// parsePackage takes a list of patterns such as a directory, and parses them.
func (g *Generator) parsePackage(ctx context.Context, patterns ...string) error {
	cfg := &packages.Config{
		// Just accept the fact we need these flags for what we want. Feel free to add
		// more, it'll just increase the time it takes to parse.
		Mode: packages.NeedTypes | packages.NeedName | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedSyntax,
		Tests:   false,
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return xerrors.Errorf("load package: %w", err)
	}

	// Only support 1 package for now. We can expand it if we need later, we
	// just need to hook up multiple packages in the generator.
	if len(pkgs) != 1 {
		return xerrors.Errorf("expected 1 package, found %d", len(pkgs))
	}

	g.pkg = pkgs[0]
	return nil
}

// generateAll will generate for all types found in the pkg
func (g *Generator) generateAll() (*TypescriptTypes, error) {
	structs := make(map[string]string)
	enums := make(map[string]types.Object)
	enumConsts := make(map[string][]*types.Const)
	//constants := make(map[string]string)

	// Look for comments that indicate to ignore a type for typescript generation.
	ignoredTypes := make(map[string]struct{})
	ignoreRegex := regexp.MustCompile("@typescript-ignore[:]?(?P<ignored_types>.*)")
	for _, file := range g.pkg.Syntax {
		for _, comment := range file.Comments {
			for _, line := range comment.List {
				text := line.Text
				matches := ignoreRegex.FindStringSubmatch(text)
				ignored := ignoreRegex.SubexpIndex("ignored_types")
				if len(matches) >= ignored && matches[ignored] != "" {
					arr := strings.Split(matches[ignored], ",")
					for _, s := range arr {
						ignoredTypes[strings.TrimSpace(s)] = struct{}{}
					}
				}

			}
		}
	}

	for _, n := range g.pkg.Types.Scope().Names() {
		obj := g.pkg.Types.Scope().Lookup(n)
		if obj == nil || obj.Type() == nil {
			// This would be weird, but it is if the package does not have the type def.
			continue
		}

		// Exclude ignored types
		if _, ok := ignoredTypes[obj.Name()]; ok {
			continue
		}

		switch obj.(type) {
		// All named types are type declarations
		case *types.TypeName:
			named, ok := obj.Type().(*types.Named)
			if !ok {
				panic("all typename should be named types")
			}
			switch named.Underlying().(type) {
			case *types.Struct:
				// Structs are obvious
				st := obj.Type().Underlying().(*types.Struct)
				codeBlock, err := g.buildStruct(obj, st)
				if err != nil {
					return nil, xerrors.Errorf("generate %q: %w", obj.Name())
				}
				structs[obj.Name()] = codeBlock
			case *types.Basic:
				// These are enums. Store to expand later.
				enums[obj.Name()] = obj
			}
		case *types.Var:
			// TODO: Are any enums var declarations? This is also codersdk.Me.
			v := obj.(*types.Var)
			var _ = v
		case *types.Const:
			c := obj.(*types.Const)
			// We only care about named constant types, since they are enums
			if named, ok := c.Type().(*types.Named); ok {
				name := named.Obj().Name()
				enumConsts[name] = append(enumConsts[name], c)
			}
		case *types.Func:
			// Noop
		default:
			fmt.Println(obj.Name())
		}
	}

	// Write all enums
	enumCodeBlocks := make(map[string]string)
	for name, v := range enums {
		var values []string
		for _, elem := range enumConsts[name] {
			// TODO: If we have non string constants, we need to handle that
			//		here.
			values = append(values, elem.Val().String())
		}
		var s strings.Builder
		s.WriteString(g.posLine(v))
		s.WriteString(fmt.Sprintf("export type %s = %s\n",
			name, strings.Join(values, " | "),
		))

		enumCodeBlocks[name] = s.String()
	}

	return &TypescriptTypes{
		Types: structs,
		Enums: enumCodeBlocks,
	}, nil
}

func (g *Generator) posLine(obj types.Object) string {
	file := g.pkg.Fset.File(obj.Pos())
	position := file.Position(obj.Pos())
	position.Filename = filepath.Join("codersdk", filepath.Base(position.Filename))
	return fmt.Sprintf("// From %s\n",
		position.String(),
	)
}

// buildStruct just prints the typescript def for a type.
func (g *Generator) buildStruct(obj types.Object, st *types.Struct) (string, error) {
	var s strings.Builder
	s.WriteString(g.posLine(obj))

	s.WriteString(fmt.Sprintf("export interface %s {\n", obj.Name()))
	// For each field in the struct, we print 1 line of the typescript interface
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))

		// Use the json name if present
		jsonName := tag.Get("json")
		arr := strings.Split(jsonName, ",")
		jsonName = arr[0]
		if jsonName == "" {
			jsonName = field.Name()
		}

		var tsType TypescriptType
		// If a `typescript:"string"` exists, we take this, and do not try to infer.
		typescriptTag := tag.Get("typescript")
		if typescriptTag == "-" {
			// Ignore this field
			continue
		} else if typescriptTag != "" {
			tsType.ValueType = typescriptTag
		} else {
			var err error
			tsType, err = g.typescriptType(field.Type())
			if err != nil {
				return "", xerrors.Errorf("typescript type: %w", err)
			}
		}

		if tsType.Comment != "" {
			s.WriteString(fmt.Sprintf("\t// %s\n", tsType.Comment))
		}
		optional := ""
		if tsType.Optional {
			optional = "?"
		}
		s.WriteString(fmt.Sprintf("\treadonly %s%s: %s\n", jsonName, optional, tsType.ValueType))
	}
	s.WriteString("}\n")
	return s.String(), nil
}

type TypescriptType struct {
	ValueType string
	Comment   string
	// Optional indicates the value is an optional field in typescript.
	Optional bool
}

// typescriptType this function returns a typescript type for a given
// golang type.
// Eg:
//	[]byte returns "string"
func (g *Generator) typescriptType(ty types.Type) (TypescriptType, error) {
	switch ty.(type) {
	case *types.Basic:
		bs := ty.(*types.Basic)
		// All basic literals (string, bool, int, etc).
		// TODO: Actually ensure the golang names are ok, otherwise,
		//		we want to put another switch to capture these types
		//		and rename to typescript.
		switch {
		case bs.Info()&types.IsNumeric > 0:
			return TypescriptType{ValueType: "number"}, nil
		case bs.Info()&types.IsBoolean > 0:
			return TypescriptType{ValueType: "boolean"}, nil
		case bs.Kind() == types.Byte:
			// TODO: @emyrk What is a byte for typescript? A string? A uint8?
			return TypescriptType{ValueType: "number", Comment: "This is a byte in golang"}, nil
		default:
			return TypescriptType{ValueType: bs.Name()}, nil
		}
	case *types.Struct:
		// TODO: This kinda sucks right now. It just dumps the struct def
		return TypescriptType{ValueType: ty.String(), Comment: "Unknown struct, this might not work"}, nil
	case *types.Map:
		// TODO: Typescript dictionary??? Object?
		// map[string][string] -> Record<string, string>
		m := ty.(*types.Map)
		keyType, err := g.typescriptType(m.Key())
		if err != nil {
			return TypescriptType{}, xerrors.Errorf("map key: %w", err)
		}
		valueType, err := g.typescriptType(m.Elem())
		if err != nil {
			return TypescriptType{}, xerrors.Errorf("map key: %w", err)
		}

		return TypescriptType{
			ValueType: fmt.Sprintf("Record<%s, %s>", keyType.ValueType, valueType.ValueType),
		}, nil
	case *types.Slice, *types.Array:
		// Slice/Arrays are pretty much the same.
		type hasElem interface {
			Elem() types.Type
		}

		arr := ty.(hasElem)
		switch {
		// When type checking here, just use the string. You can cast it
		// to a types.Basic and get the kind if you want too :shrug:
		case arr.Elem().String() == "byte":
			// All byte arrays are strings on the typescript.
			// Is this ok?
			return TypescriptType{ValueType: "string"}, nil
		default:
			// By default, just do an array of the underlying type.
			underlying, err := g.typescriptType(arr.Elem())
			if err != nil {
				return TypescriptType{}, xerrors.Errorf("array: %w", err)
			}
			return TypescriptType{ValueType: underlying.ValueType + "[]", Comment: underlying.Comment}, nil
		}
	case *types.Named:
		n := ty.(*types.Named)
		// First see if the type is defined elsewhere. If it is, we can just
		// put the name as it will be defined in the typescript codeblock
		// we generate.
		name := n.Obj().Name()
		if obj := g.pkg.Types.Scope().Lookup(name); obj != nil {
			// Sweet! Using other typescript types as fields. This could be an
			// enum or another struct
			return TypescriptType{ValueType: name}, nil
		}

		// These are special types that we handle uniquely.
		switch n.String() {
		case "net/url.URL":
			return TypescriptType{ValueType: "string"}, nil
		case "time.Time":
			// We really should come up with a standard for time.
			return TypescriptType{ValueType: "string"}, nil
		case "database/sql.NullTime":
			return TypescriptType{ValueType: "string", Optional: true}, nil
		case "github.com/google/uuid.NullUUID":
			return TypescriptType{ValueType: "string", Optional: true}, nil
		}

		// If it's a struct, just use the name of the struct type
		if _, ok := n.Underlying().(*types.Struct); ok {
			return TypescriptType{ValueType: "any", Comment: fmt.Sprintf("Named type %q unknown, using \"any\"", n.String())}, nil
		}

		// Defer to the underlying type.
		return g.typescriptType(ty.Underlying())
	case *types.Pointer:
		// Dereference pointers.
		// TODO: Nullable fields? We could say these fields can be null in the
		//		typescript.
		pt := ty.(*types.Pointer)
		resp, err := g.typescriptType(pt.Elem())
		if err != nil {
			return TypescriptType{}, xerrors.Errorf("pointer: %w", err)
		}
		resp.Optional = true
		return resp, nil
	}

	// These are all the other types we need to support.
	// time.Time, uuid, etc.
	return TypescriptType{}, xerrors.Errorf("unknown type: %s", ty.String())
}
