// Copyright 2010 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// MockGen generates mock implementations of Go interfaces.
package main

// TODO: This does not support recursive embedded interfaces.
// TODO: This does not support embedding package-local interfaces in a separate file.

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/token"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/mod/modfile"
	toolsimports "golang.org/x/tools/imports"

	"github.com/canonical/gomock/mockgen/model"
)

const (
	gomockImportPath = "github.com/canonical/gomock/gomock"

	// maxTypedArgs and maxTypedReturns are the bounds of the pre-generated
	// generic Call wrappers in the gomock package (see typed_calls.go).
	maxTypedArgs    = 8
	maxTypedReturns = 5
)

var (
	version = ""
	commit  = "none"
	date    = "unknown"
)

var (
	destination            = flag.String("destination", "", "Output file; defaults to stdout.")
	mockNames              = flag.String("mock_names", "", "Comma-separated interfaceName=mockName pairs of explicit mock names to use. Mock names default to 'Mock'+ interfaceName suffix.")
	packageOut             = flag.String("package", "", "Package of the generated code; defaults to the package of the input with a 'mock_' prefix.")
	selfPackage            = flag.String("self_package", "", "The full package import path for the generated code. The purpose of this flag is to prevent import cycles in the generated code by trying to include its own package. This can happen if the mock's package is set to one of its inputs (usually the main one) and the output is stdio so mockgen cannot detect the final output package. Setting this flag will then tell mockgen which import to exclude.")
	writeCmdComment        = flag.Bool("write_command_comment", true, "Writes the command used as a comment if true.")
	writePkgComment        = flag.Bool("write_package_comment", true, "Writes package documentation comment (godoc) if true.")
	writeSourceComment     = flag.Bool("write_source_comment", true, "Writes interface names (package mode) comment if true.")
	writeGenerateDirective = flag.Bool("write_generate_directive", false, "Add //go:generate directive to regenerate the mock")
	copyrightFile          = flag.String("copyright_file", "", "Copyright file used to add copyright header")
	buildConstraint        = flag.String("build_constraint", "", "If non-empty, added as //go:build <constraint>")
	excludeInterfaces      = flag.String("exclude_interfaces", "", "Comma-separated names of interfaces to be excluded")
	debugParser            = flag.Bool("debug_parser", false, "Print out parser results only.")
	showVersion            = flag.Bool("version", false, "Print version.")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	notifyAboutDeprecatedFlags()

	if *showVersion {
		printVersion()
		return
	}

	var pkg *model.Package
	var err error
	var packageName string

	checkArgsPackage()
	packageName = flag.Arg(0)
	interfaces := strings.Split(flag.Arg(1), ",")

	if packageName == "." {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Get current directory failed: %v", err)
		}
		packageName, err = packageNameOfDir(dir)
		if err != nil {
			log.Fatalf("Parse package name failed: %v", err)
		}

	}
	parser := packageModeParser{}
	pkg, err = parser.parsePackage(packageName, interfaces)

	if err != nil {
		log.Fatalf("Loading input failed: %v", err)
	}

	if *debugParser {
		pkg.Print(os.Stdout)
		return
	}

	outputPackageName := *packageOut
	if outputPackageName == "" {
		// pkg.Name in package mode is the base name of the import path,
		// which might have characters that are illegal to have in package names.
		outputPackageName = "mock_" + sanitize(pkg.Name)
	}

	// outputPackagePath represents the fully qualified name of the package of
	// the generated code. Its purposes are to prevent the module from importing
	// itself and to prevent qualifying type names that come from its own
	// package (i.e. if there is a type called X then we want to print "X" not
	// "package.X" since "package" is this package). This can happen if the mock
	// is output into an already existing package.
	outputPackagePath := *selfPackage
	if outputPackagePath == "" && *destination != "" {
		dstPath, err := filepath.Abs(filepath.Dir(*destination))
		if err == nil {
			pkgPath, err := parsePackageImport(dstPath)
			if err == nil {
				outputPackagePath = pkgPath
			} else {
				log.Println("Unable to infer -self_package from destination file path:", err)
			}
		} else {
			log.Println("Unable to determine destination file path:", err)
		}
	}

	g := &generator{
		buildConstraint: *buildConstraint,
	}
	g.srcPackage = packageName
	g.srcInterfaces = flag.Arg(1)
	g.destination = *destination

	if *mockNames != "" {
		g.mockNames = parseMockNames(*mockNames)
	}
	if *copyrightFile != "" {
		header, err := os.ReadFile(*copyrightFile)
		if err != nil {
			log.Fatalf("Failed reading copyright file: %v", err)
		}

		g.copyrightHeader = string(header)
	}
	if err := g.Generate(pkg, outputPackageName, outputPackagePath); err != nil {
		log.Fatalf("Failed generating mock: %v", err)
	}
	output := g.Output()
	dst := os.Stdout
	if len(*destination) > 0 {
		if err := os.MkdirAll(filepath.Dir(*destination), os.ModePerm); err != nil {
			log.Fatalf("Unable to create directory: %v", err)
		}
		existing, err := os.ReadFile(*destination)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("Failed reading pre-exiting destination file: %v", err)
		}
		if len(existing) == len(output) && bytes.Equal(existing, output) {
			return
		}
		f, err := os.Create(*destination)
		if err != nil {
			log.Fatalf("Failed opening destination file: %v", err)
		}
		defer f.Close()
		dst = f
	}
	if _, err := dst.Write(output); err != nil {
		log.Fatalf("Failed writing to destination: %v", err)
	}
}

func parseMockNames(names string) map[string]string {
	mocksMap := make(map[string]string)
	for _, kv := range strings.Split(names, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 || parts[1] == "" {
			log.Fatalf("bad mock names spec: %v", kv)
		}
		mocksMap[parts[0]] = parts[1]
	}
	return mocksMap
}

func parseExcludeInterfaces(names string) map[string]struct{} {
	splitNames := strings.Split(names, ",")
	namesSet := make(map[string]struct{}, len(splitNames))
	for _, name := range splitNames {
		if name == "" {
			continue
		}

		namesSet[name] = struct{}{}
	}

	if len(namesSet) == 0 {
		return nil
	}

	return namesSet
}

func checkArgsPackage() {
	if flag.NArg() != 2 {
		usage()
		log.Fatal("Expected exactly two arguments")
	}
}

func usage() {
	_, _ = io.WriteString(os.Stderr, usageText)
	flag.PrintDefaults()
}

const usageText = `mockgen generates mock implementations of Go interfaces.

Package mode works by specifying the package and interface names.
It requires two non-flag arguments: an import path, and a
comma-separated list of symbols.
You can use "." to refer to the current path's package.
Example:
	mockgen database/sql/driver Conn,Driver
	mockgen . SomeInterface

`

type generator struct {
	buf                       bytes.Buffer
	indent                    string
	mockNames                 map[string]string // may be empty
	destination               string            // may be empty
	srcPackage, srcInterfaces string            // may be empty
	copyrightHeader           string
	buildConstraint           string // may be empty

	packageMap map[string]string // map from import path to package name
}

func (g *generator) p(format string, args ...any) {
	_, _ = fmt.Fprintf(&g.buf, g.indent+format+"\n", args...)
}

func (g *generator) in() {
	g.indent += "\t"
}

func (g *generator) out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[0 : len(g.indent)-1]
	}
}

// gomockPkg returns the package qualifier for the gomock package
// (e.g. "gomock."), or an empty string when generating code inside
// the gomock package itself.
func (g *generator) gomockPkg() string {
	if pkg, ok := g.packageMap[gomockImportPath]; ok {
		return pkg + "."
	}
	return ""
}

// sanitize cleans up a string to make a suitable package name.
func sanitize(s string) string {
	t := ""
	for _, r := range s {
		if t == "" {
			if unicode.IsLetter(r) || r == '_' {
				t += string(r)
				continue
			}
		} else {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				t += string(r)
				continue
			}
		}
		t += "_"
	}
	if t == "_" {
		t = "x"
	}
	return t
}

func (g *generator) Generate(pkg *model.Package, outputPkgName string, outputPackagePath string) error {
	if outputPkgName != pkg.Name && *selfPackage == "" {
		// reset outputPackagePath if it's not passed in through -self_package
		outputPackagePath = ""
	}

	if g.copyrightHeader != "" {
		lines := strings.Split(g.copyrightHeader, "\n")
		for _, line := range lines {
			g.p("// %s", line)
		}
		g.p("")
	}

	if g.buildConstraint != "" {
		g.p("//go:build %s", g.buildConstraint)
		// https://pkg.go.dev/cmd/go#hdr-Build_constraints:~:text=a%20build%20constraint%20should%20be%20followed%20by%20a%20blank%20line
		g.p("")
	}

	g.p("// Code generated by MockGen. DO NOT EDIT.")
	if *writeSourceComment {
		g.p("// Source: %v (interfaces: %v)", g.srcPackage, g.srcInterfaces)
	}
	if *writeCmdComment {
		g.p("//")
		g.p("// Generated by this command:")
		g.p("//")
		// only log the name of the executable, not the full path
		name := filepath.Base(os.Args[0])
		if runtime.GOOS == "windows" {
			name = strings.TrimSuffix(name, ".exe")
		}
		g.p("//\t%v", strings.Join(append([]string{name}, os.Args[1:]...), " "))
		g.p("//")
	}

	// Get all required imports, and generate unique names for them all.
	im := pkg.Imports()
	im[gomockImportPath] = true

	// Sort keys to make import alias generation predictable
	sortedPaths := make([]string, len(im))
	x := 0
	for pth := range im {
		sortedPaths[x] = pth
		x++
	}
	sort.Strings(sortedPaths)

	packagesName := createPackageMap(sortedPaths)

	g.packageMap = make(map[string]string, len(im))
	localNames := make(map[string]bool, len(im))
	for _, pth := range sortedPaths {
		base, ok := packagesName[pth]
		if !ok {
			base = sanitize(path.Base(pth))
		}

		// Local names for an imported package can usually be the basename of
		// the import path. A couple of situations don't permit that, such as
		// duplicate local names (e.g. importing "html/template" and
		// "text/template"), or where the basename is a keyword
		// (e.g. "foo/case"). try base0, base1, ...
		pkgName := base

		i := 0
		for localNames[pkgName] || token.Lookup(pkgName).IsKeyword() || pkgName == "any" {
			pkgName = base + strconv.Itoa(i)
			i++
		}

		// Avoid importing package if source pkg == output pkg
		if pth == pkg.PkgPath && outputPackagePath == pkg.PkgPath {
			continue
		}

		g.packageMap[pth] = pkgName
		localNames[pkgName] = true
	}

	// Ensure there is an empty line between “generated by” block and
	// package documentation comments to follow the recommendations:
	// https://go.dev/wiki/CodeReviewComments#package-comments
	// That is, “generated by” should not be a package comment.
	g.p("")

	if *writePkgComment {
		g.p("// Package %v is a generated GoMock package.", outputPkgName)
	}
	g.p("package %v", outputPkgName)
	g.p("")
	g.p("import (")
	g.in()
	for pkgPath, pkgName := range g.packageMap {
		if pkgPath == outputPackagePath {
			continue
		}
		g.p("%v %q", pkgName, pkgPath)
	}
	for _, pkgPath := range pkg.DotImports {
		g.p(". %q", pkgPath)
	}
	g.out()
	g.p(")")

	if *writeGenerateDirective {
		g.p("//go:generate %v", strings.Join(os.Args, " "))
	}

	for _, intf := range pkg.Interfaces {
		if err := g.GenerateMockInterface(intf, outputPackagePath); err != nil {
			return err
		}
	}

	return nil
}

// The name of the mock type to use for the given interface identifier.
func (g *generator) mockName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}

	return "Mock" + typeName
}

// formattedTypeParams returns a long and short form of type param info used for
// printing. If analyzing a interface with type param [I any, O any] the result
// will be:
// "[I any, O any]", "[I, O]"
func (g *generator) formattedTypeParams(it *model.Interface, pkgOverride string) (string, string) {
	if len(it.TypeParams) == 0 {
		return "", ""
	}
	var long, short strings.Builder
	long.WriteString("[")
	short.WriteString("[")
	for i, v := range it.TypeParams {
		if i != 0 {
			long.WriteString(", ")
			short.WriteString(", ")
		}
		long.WriteString(v.Name)
		short.WriteString(v.Name)
		long.WriteString(fmt.Sprintf(" %s", v.Type.String(g.packageMap, pkgOverride)))
	}
	long.WriteString("]")
	short.WriteString("]")
	return long.String(), short.String()
}

func (g *generator) GenerateMockInterface(
	intf *model.Interface,
	outputPackagePath string,
) error {
	mockType := g.mockName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	// Reject methods that exceed the pre-generated arity bounds.
	for _, m := range intf.Methods {
		if len(m.In) > maxTypedArgs {
			return fmt.Errorf(
				"%s.%s: %d input parameters exceeds"+
					" maximum of %d",
				intf.Name, m.Name,
				len(m.In), maxTypedArgs,
			)
		}
		if len(m.Out) > maxTypedReturns {
			return fmt.Errorf(
				"%s.%s: %d return values exceeds"+
					" maximum of %d",
				intf.Name, m.Name,
				len(m.Out), maxTypedReturns,
			)
		}
	}

	g.p("")
	g.p("// %v is a mock of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	g.p("ctrl     *gomock.Controller")
	g.p("recorder *%vMockRecorder%v", mockType, shortTp)
	g.p("isgomock struct{}")
	g.out()
	g.p("}")
	g.p("")

	g.p(
		"// %vMockRecorder is the mock recorder for %v.",
		mockType, mockType,
	)
	g.p("type %vMockRecorder%v struct {", mockType, longTp)
	g.in()
	g.p("mock *%v%v", mockType, shortTp)
	// Per-method expects slices, sorted alphabetically.
	sorted := make([]*model.Method, len(intf.Methods))
	copy(sorted, intf.Methods)
	sort.Sort(byMethodName(sorted))
	for _, m := range sorted {
		fieldName := expectsFieldName(m.Name)
		callType := g.callTypeForMethod(m, outputPackagePath)
		g.p("%s []*%s", fieldName, callType)
	}
	g.out()
	g.p("}")
	g.p("")

	g.p("// New%v creates a new mock instance.", mockType)
	g.p(
		"func New%v%v(ctrl *gomock.Controller) *%v%v {",
		mockType, longTp, mockType, shortTp,
	)
	g.in()
	g.p("mock := &%v%v{ctrl: ctrl}", mockType, shortTp)
	g.p(
		"mock.recorder = &%vMockRecorder%v{mock: mock}",
		mockType, shortTp,
	)
	g.p("return mock")
	g.out()
	g.p("}")
	g.p("")

	g.p(
		"// EXPECT returns an object that allows the caller" +
			" to indicate expected use.",
	)
	g.p(
		"func (m *%v%v) EXPECT() *%vMockRecorder%v {",
		mockType, shortTp, mockType, shortTp,
	)
	g.in()
	g.p("return m.recorder")
	g.out()
	g.p("}")

	g.GenerateMockMethods(
		mockType, intf, outputPackagePath, longTp, shortTp,
	)
	return nil
}

type byMethodName []*model.Method

func (b byMethodName) Len() int           { return len(b) }
func (b byMethodName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byMethodName) Less(i, j int) bool { return b[i].Name < b[j].Name }

func (g *generator) GenerateMockMethods(mockType string, intf *model.Interface, pkgOverride, longTp, shortTp string) {
	sort.Sort(byMethodName(intf.Methods))
	for _, m := range intf.Methods {
		g.p("")
		_ = g.GenerateMockMethod(mockType, m, pkgOverride, shortTp)
		g.p("")
		_ = g.GenerateMockRecorderMethod(intf, m, pkgOverride, shortTp)
		g.p("")
		_ = g.GenerateMockReturnCallMethod(intf, m, pkgOverride, longTp, shortTp)
	}
}

func makeArgString(argNames, argTypes []string) string {
	args := make([]string, len(argNames))
	for i, name := range argNames {
		// specify the type only once for consecutive args of the same type
		if i+1 < len(argTypes) && argTypes[i] == argTypes[i+1] {
			args[i] = name
		} else {
			args[i] = name + " " + argTypes[i]
		}
	}
	return strings.Join(args, ", ")
}

func (g *generator) GenerateMockMethod(
	mockType string,
	m *model.Method,
	pkgOverride, shortTp string,
) error {
	argNames := g.getArgNames(m, true /* in */)
	argTypes := g.getArgTypes(m, pkgOverride, true /* in */)
	argString := makeArgString(argNames, argTypes)

	rets := make([]string, len(m.Out))
	for i, p := range m.Out {
		rets[i] = p.Type.String(g.packageMap, pkgOverride)
	}
	retString := strings.Join(rets, ", ")
	if len(rets) > 1 {
		retString = "(" + retString + ")"
	}
	if retString != "" {
		retString = " " + retString
	}

	ia := newIdentifierAllocator(argNames)
	idRecv := ia.allocateIdentifier("m")
	gomockPkg := g.gomockPkg()
	fieldName := expectsFieldName(m.Name)

	g.p("// %v mocks base method.", m.Name)
	g.p(
		"func (%v *%v%v) %v(%v)%v {",
		idRecv, mockType, shortTp, m.Name, argString, retString,
	)
	g.in()
	g.p("%s.ctrl.T.Helper()", idRecv)

	if m.Variadic == nil {
		var callArgs string
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}
		if len(m.Out) == 0 {
			g.p(
				`%sDispatch%d_%d(&%s.recorder.%s,`+
					` %s.ctrl, %s, "%s"%s)`,
				gomockPkg, len(m.In), len(m.Out),
				idRecv, fieldName,
				idRecv, idRecv, m.Name, callArgs,
			)
		} else {
			g.p(
				`return %sDispatch%d_%d(`+
					`&%s.recorder.%s,`+
					` %s.ctrl, %s, "%s"%s)`,
				gomockPkg, len(m.In), len(m.Out),
				idRecv, fieldName,
				idRecv, idRecv, m.Name, callArgs,
			)
		}
	} else {
		var fixedArgs string
		if len(m.In) > 0 {
			fixedArgs = ", " +
				strings.Join(
					argNames[:len(argNames)-1], ", ",
				)
		}
		varArg := argNames[len(argNames)-1]
		if len(m.Out) == 0 {
			g.p(
				`%sDispatch%dV_%d(&%s.recorder.%s,`+
					` %s.ctrl, %s, "%s"%s, %s...)`,
				gomockPkg, len(m.In), len(m.Out),
				idRecv, fieldName,
				idRecv, idRecv, m.Name, fixedArgs, varArg,
			)
		} else {
			g.p(
				`return %sDispatch%dV_%d(`+
					`&%s.recorder.%s,`+
					` %s.ctrl, %s, "%s"%s, %s...)`,
				gomockPkg, len(m.In), len(m.Out),
				idRecv, fieldName,
				idRecv, idRecv, m.Name, fixedArgs, varArg,
			)
		}
	}

	g.out()
	g.p("}")
	return nil
}

func (g *generator) GenerateMockRecorderMethod(
	intf *model.Interface,
	m *model.Method,
	pkgOverride, shortTp string,
) error {
	mockType := g.mockName(intf.Name)
	argNames := g.getArgNames(m, true)
	gomockPkg := g.gomockPkg()
	fieldName := expectsFieldName(m.Name)

	// Recorder method signature: all fixed args as "any",
	// variadic as "...any".
	var argString string
	if m.Variadic == nil {
		argString = strings.Join(argNames, ", ")
		if argString != "" {
			argString += " any"
		}
	} else {
		fixed := argNames[:len(argNames)-1]
		argString = strings.Join(fixed, ", ")
		if argString != "" {
			argString += " any, "
		}
		argString += argNames[len(argNames)-1] + " ...any"
	}

	ia := newIdentifierAllocator(argNames)
	idRecv := ia.allocateIdentifier("mr")

	g.p(
		"// %v indicates an expected call of %v.",
		m.Name, m.Name,
	)
	g.p(
		"func (%s *%vMockRecorder%v) %v(%v) *%s%sCall%s {",
		idRecv, mockType, shortTp, m.Name, argString,
		mockType, m.Name, shortTp,
	)
	g.in()
	g.p("%s.mock.ctrl.T.Helper()", idRecv)

	// Build type-argument string for NewCallN_M / NewCallNV_M.
	typeArgs := make([]string, 0, len(m.In)+len(m.Out)+1)
	for _, p := range m.In {
		typeArgs = append(
			typeArgs,
			p.Type.String(g.packageMap, pkgOverride),
		)
	}
	if m.Variadic != nil {
		typeArgs = append(
			typeArgs,
			m.Variadic.Type.String(g.packageMap, pkgOverride),
		)
	}
	for _, p := range m.Out {
		typeArgs = append(
			typeArgs,
			p.Type.String(g.packageMap, pkgOverride),
		)
	}
	var typeArgStr string
	if len(typeArgs) > 0 {
		typeArgStr = "[" + strings.Join(typeArgs, ", ") + "]"
	}

	if m.Variadic == nil {
		var matcherArgs string
		if len(argNames) > 0 {
			parts := make([]string, len(argNames))
			for i, n := range argNames {
				parts[i] = gomockPkg +
					"EnsureMatcher(" + n + ")"
			}
			matcherArgs = ", " + strings.Join(parts, ", ")
		}
		g.p(
			`call := %sNewCall%d_%d%s(`+
				`%s.mock.ctrl.T, %s.mock, "%s"%s)`,
			gomockPkg, len(m.In), len(m.Out), typeArgStr,
			idRecv, idRecv, m.Name, matcherArgs,
		)
	} else {
		// Variadic: build varArgs slice, then NewCallNV_M.
		varName := argNames[len(argNames)-1]
		idVarArgs := ia.allocateIdentifier("varArgs")
		idI := ia.allocateIdentifier("i")
		idA := ia.allocateIdentifier("a")
		g.p(
			"%s := make([]%sMatcher, len(%s))",
			idVarArgs, gomockPkg, varName,
		)
		g.p(
			"for %s, %s := range %s {",
			idI, idA, varName,
		)
		g.in()
		g.p(
			"%s[%s] = %sEnsureMatcher(%s)",
			idVarArgs, idI, gomockPkg, idA,
		)
		g.out()
		g.p("}")
		var fixedMatcherArgs string
		if len(m.In) > 0 {
			parts := make([]string, len(m.In))
			for i, n := range argNames[:len(m.In)] {
				parts[i] = gomockPkg +
					"EnsureMatcher(" + n + ")"
			}
			fixedMatcherArgs = ", " +
				strings.Join(parts, ", ")
		}
		g.p(
			`call := %sNewCall%dV_%d%s(`+
				`%s.mock.ctrl.T, %s.mock, "%s"%s, %s)`,
			gomockPkg, len(m.In), len(m.Out), typeArgStr,
			idRecv, idRecv, m.Name,
			fixedMatcherArgs, idVarArgs,
		)
	}

	g.p(
		"%s.%s = append(%s.%s, call)",
		idRecv, fieldName, idRecv, fieldName,
	)
	g.p("%s.mock.ctrl.Track(call.Call)", idRecv)
	g.p("return call")
	g.out()
	g.p("}")
	return nil
}

func (g *generator) GenerateMockReturnCallMethod(
	intf *model.Interface,
	m *model.Method,
	pkgOverride, longTp, shortTp string,
) error {
	mockType := g.mockName(intf.Name)
	return g.generateTypedCallAlias(
		mockType, m, pkgOverride, longTp,
	)
}

// generateTypedCallAlias emits a single-line type alias, e.g.:
//
//	type MockFooBarCall = gomock.Call1_1[string, error]
//	type MockFooFooCall = gomock.Call1V_1[string, int, string]
func (g *generator) generateTypedCallAlias(
	mockType string,
	m *model.Method,
	pkgOverride, longTp string,
) error {
	gomockPkg := g.gomockPkg()
	typeArgs := make([]string, 0, len(m.In)+len(m.Out)+1)
	var callTypeName string
	if m.Variadic == nil {
		for _, p := range m.In {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		for _, p := range m.Out {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		callTypeName = fmt.Sprintf(
			"Call%d_%d", len(m.In), len(m.Out),
		)
	} else {
		for _, p := range m.In {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		typeArgs = append(
			typeArgs,
			m.Variadic.Type.String(g.packageMap, pkgOverride),
		)
		for _, p := range m.Out {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		callTypeName = fmt.Sprintf(
			"Call%dV_%d", len(m.In), len(m.Out),
		)
	}
	var typeArgStr string
	if len(typeArgs) > 0 {
		typeArgStr = "[" + strings.Join(typeArgs, ", ") + "]"
	}
	g.p(
		"// %s%sCall is the typed call wrapper for %s.",
		mockType, m.Name, m.Name,
	)
	g.p(
		"type %s%sCall%s = %s%s%s",
		mockType, m.Name, longTp,
		gomockPkg, callTypeName, typeArgStr,
	)
	return nil
}

// lowerFirst returns s with its first rune lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// expectsFieldName returns the recorder field name for a method,
// e.g. "Bar" → "barExpects".
func expectsFieldName(methodName string) string {
	return lowerFirst(methodName) + "Expects"
}

// callTypeForMethod returns the gomock CallN_M or CallNV_M type string
// (without leading "*") for use in recorder field declarations and
// type aliases.
func (g *generator) callTypeForMethod(
	m *model.Method, pkgOverride string,
) string {
	gomockPkg := g.gomockPkg()
	typeArgs := make([]string, 0, len(m.In)+len(m.Out)+1)
	var name string
	if m.Variadic == nil {
		for _, p := range m.In {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		for _, p := range m.Out {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		name = fmt.Sprintf("Call%d_%d", len(m.In), len(m.Out))
	} else {
		for _, p := range m.In {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		typeArgs = append(
			typeArgs,
			m.Variadic.Type.String(g.packageMap, pkgOverride),
		)
		for _, p := range m.Out {
			typeArgs = append(
				typeArgs,
				p.Type.String(g.packageMap, pkgOverride),
			)
		}
		name = fmt.Sprintf("Call%dV_%d", len(m.In), len(m.Out))
	}
	if len(typeArgs) == 0 {
		return gomockPkg + name
	}
	return gomockPkg + name +
		"[" + strings.Join(typeArgs, ", ") + "]"
}

// nameExistsAsPackage returns true if the name exists as a package name.
// This is used to avoid name collisions when generating mock method arguments.
func (g *generator) nameExistsAsPackage(name string) bool {
	for _, symbolName := range g.packageMap {
		if symbolName == name {
			return true
		}
	}
	return false
}

func (g *generator) getArgNames(m *model.Method, in bool) []string {
	var params []*model.Parameter
	if in {
		params = m.In
	} else {
		params = m.Out
	}
	argNames := make([]string, len(params))

	for i, p := range params {
		name := p.Name

		if name == "" || name == "_" || g.nameExistsAsPackage(name) {
			name = fmt.Sprintf("arg%d", i)
		}
		argNames[i] = name
	}
	if m.Variadic != nil && in {
		name := m.Variadic.Name

		if name == "" || g.nameExistsAsPackage(name) {
			name = fmt.Sprintf("arg%d", len(params))
		}
		argNames = append(argNames, name)
	}
	return argNames
}

func (g *generator) getArgTypes(m *model.Method, pkgOverride string, in bool) []string {
	var params []*model.Parameter
	if in {
		params = m.In
	} else {
		params = m.Out
	}
	argTypes := make([]string, len(params))
	for i, p := range params {
		argTypes[i] = p.Type.String(g.packageMap, pkgOverride)
	}
	if m.Variadic != nil {
		argTypes = append(argTypes, "..."+m.Variadic.Type.String(g.packageMap, pkgOverride))
	}
	return argTypes
}

type identifierAllocator map[string]struct{}

func newIdentifierAllocator(taken []string) identifierAllocator {
	a := make(identifierAllocator, len(taken))
	for _, s := range taken {
		a[s] = struct{}{}
	}
	return a
}

func (o identifierAllocator) allocateIdentifier(want string) string {
	id := want
	for i := 2; ; i++ {
		if _, ok := o[id]; !ok {
			o[id] = struct{}{}
			return id
		}
		id = want + "_" + strconv.Itoa(i)
	}
}

// Output returns the generator's output, formatted in the standard Go style.
func (g *generator) Output() []byte {
	src, err := toolsimports.Process(g.destination, g.buf.Bytes(), nil)
	if err != nil {
		log.Fatalf("Failed to format generated source code: %s\n%s", err, g.buf.String())
	}
	return src
}

// createPackageMap returns a map of import path to package name
// for specified importPaths.
func createPackageMap(importPaths []string) map[string]string {
	var pkg struct {
		Name       string
		ImportPath string
	}
	pkgMap := make(map[string]string)
	b := bytes.NewBuffer(nil)
	args := []string{"list", "-json=ImportPath,Name"}
	args = append(args, importPaths...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = b
	cmd.Run()
	dec := json.NewDecoder(b)
	for dec.More() {
		err := dec.Decode(&pkg)
		if err != nil {
			log.Printf("failed to decode 'go list' output: %v", err)
			continue
		}
		pkgMap[pkg.ImportPath] = pkg.Name
	}
	return pkgMap
}

func printVersion() {
	if version != "" {
		fmt.Printf("v%s\nCommit: %s\nDate: %s\n", version, commit, date)
	} else {
		printModuleVersion()
	}
}

var errOutsideGoPath = errors.New(
	"source directory is outside GOPATH",
)

// packageNameOfDir returns the import path of the package in srcDir.
func packageNameOfDir(srcDir string) (string, error) {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		log.Fatal(err)
	}

	var goFilePath string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			goFilePath = file.Name()
			break
		}
	}
	if goFilePath == "" {
		return "", fmt.Errorf("go source file not found %s", srcDir)
	}
	return parsePackageImport(srcDir)
}

// parseImportPackage get package import path via source file
// an alternative implementation is to use:
// cfg := &packages.Config{Mode: packages.NeedName, Tests: true, Dir: srcDir}
// pkgs, err := packages.Load(cfg, "file="+source)
// However, it will call "go list" and slow down the performance
func parsePackageImport(srcDir string) (string, error) {
	moduleMode := os.Getenv("GO111MODULE")
	// trying to find the module
	if moduleMode != "off" {
		currentDir := srcDir
		for {
			dat, err := os.ReadFile(filepath.Join(currentDir, "go.mod"))
			if os.IsNotExist(err) {
				if currentDir == filepath.Dir(currentDir) {
					// at the root
					break
				}
				currentDir = filepath.Dir(currentDir)
				continue
			} else if err != nil {
				return "", err
			}
			modulePath := modfile.ModulePath(dat)
			return filepath.ToSlash(filepath.Join(modulePath, strings.TrimPrefix(srcDir, currentDir))), nil
		}
	}
	// fall back to GOPATH mode
	goPaths := os.Getenv("GOPATH")
	if goPaths == "" {
		return "", fmt.Errorf("GOPATH is not set")
	}
	goPathList := strings.Split(goPaths, string(os.PathListSeparator))
	for _, goPath := range goPathList {
		sourceRoot := filepath.Join(goPath, "src") + string(os.PathSeparator)
		if strings.HasPrefix(srcDir, sourceRoot) {
			return filepath.ToSlash(strings.TrimPrefix(srcDir, sourceRoot)), nil
		}
	}
	return "", errOutsideGoPath
}
