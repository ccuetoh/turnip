package turnip

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

var ErrUnsupportedType = errors.New("unsupported type")

const (
	jsonIgnoreTag = "-"
)

type Resolver interface {
	ResolveJSON(res gjson.Result) (reflect.Type, error)
}

type traverseResolver struct {
	env    environment
	logger *zap.SugaredLogger
	paths  map[*candidate]jsonPaths
}

// TODO Return multiple posibilities
func (r *traverseResolver) ResolveJSON(res gjson.Result) (reflect.Type, error) {
	for c, paths := range r.paths {
		for path, typ := range paths {
			if res.Get(path).Type == typ {
				return c.typ, nil
			}
		}
	}

	return nil, nil
}

type jsonPaths map[string]gjson.Type

func newTraverseResolver(env environment) (*traverseResolver, error) {
	r := &traverseResolver{
		env:    env,
		logger: env.logger.Named("traverse-resolver"),
	}

	r.logger.Infow("building paths", zap.Int("candidates", len(env.candidates)))

	candidatePaths := make(map[*candidate]jsonPaths, len(env.candidates))
	for _, c := range env.candidates {
		paths, err := buildPathsForRoot(c.typ)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.typ, err)
		}

		r.logger.Infof("built %d paths for %s:", len(paths), c.typ)
		for path, t := range paths {
			r.logger.Infof("  %s -> %s", path, t.String())
		}

		candidatePaths[c] = paths
	}

	r.logger.Info("finding paths to use as fingerprints")

	r.paths = makeUniquePaths(candidatePaths)

	for c, paths := range r.paths {
		r.logger.Infof("%s:", c.typ)
		for path, typ := range paths {
			r.logger.Infof("  %s -> %s", path, typ.String())
		}
	}

	return r, nil
}

func buildPathsForRoot(t reflect.Type) (jsonPaths, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New("not a struct")
	}

	paths := make(jsonPaths, t.NumField())

	for _, f := range reflect.VisibleFields(t) {
		if !f.IsExported() {
			continue
		}

		err := buildPathsForField(paths, appendToPath("", getJSONName(f)), f.Type)
		if err != nil {
			return nil, fmt.Errorf("%s :%w", f.Name, err)
		}
	}

	return paths, nil
}

func buildPathsForField(paths jsonPaths, curr string, t reflect.Type) error {
	jsonType, err := getJSONType(t)
	if err != nil {
		return err
	}

	if jsonType == gjson.True || jsonType == gjson.False {
		// Booleans are constants in JSON, but a type in Go. We don't care about what value it has, just the type, so
		// we'll accept either constant True or False
		paths[curr] = gjson.True
		paths[curr] = gjson.False
		return nil
	}

	if jsonType != gjson.JSON {
		paths[curr] = jsonType
		return nil
	}

	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
		// We can't validate the type yet, since JSON does not distinction between all of this. We'll give the parser
		// the final say
		paths[curr] = gjson.JSON
		return nil
	}

	for _, f := range reflect.VisibleFields(t) {
		if !f.IsExported() {
			continue
		}

		name := getJSONName(f)
		if name == jsonIgnoreTag {
			continue
		}

		err = buildPathsForField(paths, appendToPath(curr, name), f.Type)
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO Use a cache of encountered paths instead
func makeUniquePaths(candidatePaths map[*candidate]jsonPaths) map[*candidate]jsonPaths {
	// This is a very expensive operation, but we only do it once at the creation of the resolver
	for candidateA, pathsA := range candidatePaths {
		for pathA, typeA := range pathsA {
			// We go thorough all candidates again, searching for duplicates
			for candidateB, pathsB := range candidatePaths {
				if candidateA.typ == candidateB.typ {
					// Same candidate
					continue
				}

				for pathB, typeB := range pathsB {
					if pathA == pathB && typeA == typeB {
						// This can end up deleting pathA multiple times, which is no-op
						// That's fine since we want to delete dupes from all maps, not just the first one
						delete(pathsA, pathA)
						delete(pathsB, pathB)
						break
					}
				}
			}
		}
	}

	return candidatePaths
}

func getJSONName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == jsonIgnoreTag {
		return jsonIgnoreTag
	}

	return f.Name
}

func getJSONType(t reflect.Type) (gjson.Type, error) {
	switch t.Kind() {
	case reflect.String:
		return gjson.String, nil
	case reflect.Bool:
		return gjson.True, nil
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return gjson.Number, nil
	case reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.Struct,
		reflect.Array:
		return gjson.JSON, nil
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer,
		reflect.Invalid:
		return -1, fmt.Errorf("%w: %s", ErrUnsupportedType, t.String())
	default:
		return -1, fmt.Errorf("unknown type: %s", t.String())
	}
}

func appendToPath(path, name string) string {
	name = normalizeName(name)
	if len(path) == 0 || strings.HasSuffix(path, ".") {
		return path + name
	}

	return path + "." + name
}

func normalizeName(name string) string {
	const cutset = " _-"
	return cutsetString(strings.ToLower(name), cutset)
}

func cutsetString(s, cutset string) string {
	for _, c := range strings.Split(cutset, "") {
		s = strings.ReplaceAll(s, c, "")
	}

	return s
}
