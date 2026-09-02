Here is a complete breakdown of how hestia service generate works, from CLI invocation to the final registrations.go output.
Relevant File Paths
Layer	File
CLI entry point	/home/augustine/projects/hestia/cmd/hestia/main.go
CLI command definition (service generate)	/home/augustine/projects/hestia/cmd/hestia/core/service.go
Core generation orchestrator	/home/augustine/projects/hestia/cmd/hestia/core/gen/gen.go
Code rendering (registrations.go, policies.go, collector)	/home/augustine/projects/hestia/cmd/hestia/core/gen/render.go
Annotation parser (@hestia.register)	/home/augustine/projects/hestia/cmd/hestia/core/annotate/annotate.go
Project config (hestia.json)	/home/augustine/projects/hestia/cmd/hestia/core/config.go
Module scaffold	/home/augustine/projects/hestia/cmd/hestia/core/module.go
Annotate tests	/home/augustine/projects/hestia/cmd/hestia/core/annotate/annotate_test.go
Gen tests	/home/augustine/projects/hestia/cmd/hestia/core/gen/gen_test.go
CLI integration tests	/home/augustine/projects/hestia/cmd/hestia/core/service_test.go
Test fixtures	/home/augustine/projects/hestia/cmd/hestia/core/gen/testdata/users/service.go
Test fixtures (annotate)	/home/augustine/projects/hestia/cmd/hestia/core/annotate/testdata/users/service.go
Test fixtures (streaming)	/home/augustine/projects/hestia/cmd/hestia/core/annotate/testdata/feed/service.go
Test fixtures (bad)	/home/augustine/projects/hestia/cmd/hestia/core/annotate/testdata/bad/service.go
Test fixtures (deprecated header_fields)	/home/augustine/projects/hestia/cmd/hestia/core/annotate/testdata/bad_header_fields/service.go
Golden output	/home/augustine/projects/hestia/cmd/hestia/core/gen/testdata/users/registrations.go.golden
Real auth service (source annotations)	/home/augustine/projects/hestia/core/system/auth/service.go
Real auth registrations (generated output)	/home/augustine/projects/hestia/core/system/auth/registrations.go
1. CLI Command: hestia service generate
Defined in /home/augustine/projects/hestia/cmd/hestia/core/service.go, lines 296-375.
The command is registered as a subcommand of hestia service (line 33):
ServiceCmd.AddCommand(serviceGenerateCmd)
Two modes:
- Single service: hestia service generate <module> <name> -- resolves the module directory, calls gen.Generate(dir), then gen.SeedSanitization(dir), and refreshCollector(moduleDir).
- All services: hestia service generate --all -- iterates every module directory, finds subdirectories that already contain registrations.go, and runs gen.Generate + gen.SeedSanitization + refreshCollector on each.
The key call chain (single-service path, lines 350-373):
serviceModuleDir(module)   -> resolve module directory
gen.Generate(dir)          -> parse annotations + render registrations.go + policies.go
gen.SeedSanitization(dir)  -> seed sanitization.go (write-once)
refreshCollector(moduleDir) -> regenerate services.go + gen_sanitization.go collectors
2. How Handler Methods Are Discovered
Step 2a: The annotate parser finds @hestia.register blocks
File: /home/augustine/projects/hestia/cmd/hestia/core/annotate/annotate.go, Parse() function (lines 117-157).
The discovery logic:
1. Parse all .go files in the service directory (skipping _test.go) using go/parser with parser.ParseComments.
2. Iterate all declarations in each file, looking for *ast.FuncDecl nodes that have:
- A non-nil doc comment (fd.Doc != nil)
- A receiver (fd.Recv != nil)
3. Receiver type must end with "Service" -- receiverService() (lines 175-192) extracts the receiver type name and checks strings.HasSuffix(id.Name, "Service"). This means only methods on *AuthService, *UsersService, etc. are considered.
4. Extract @hestia.register(...) blocks from the doc comment -- annotationBlocks() (lines 198-237) scans the comment text for @hestia.register, then reads balanced parentheses to extract the key="value" pairs inside.
5. Build a structured Annotation from each block via buildAnnotation() (lines 288-374).
Step 2b: The gen layer uses annotations to generate code
File: /home/augustine/projects/hestia/cmd/hestia/core/gen/gen.go, Generate() (lines 29-56):
func Generate(dir string) error {
    anns, err := annotate.Parse(dir)    // Step 2a: find all annotations
    pkg, err := packageName(dir)        // read "package X" from .go files
    regSrc, err := renderRegistrations(pkg, anns)  // render registrations.go
    writeFormatted(filepath.Join(dir, "registrations.go"), regSrc)
    polSrc, err := renderPolicies(pkg, anns)       // render policies.go
    writeFormatted(filepath.Join(dir, "policies.go"), polSrc)
}
3. How Internal, BootstrapSafe, Enabled, Intent, Input, Output Are Determined
All of this happens in buildAnnotation() (annotate.go lines 288-374) and writeRegistration() (render.go lines 102-160).
From annotation attributes (doc-comment):
Field	Source	Parse location
MessageName	name="system:auth:session:create"	attrs["name"] (line 300)
Intent (maps to abstract.Verb)	intent="create"	attrs["intent"] (line 301), validated against the Verb enum (lines 314-317)
Rule (policy rule key)	rule="administrator"	attrs["rule"] (line 302), defaults to "authenticated" in render (render.go line 231)
Description	description="..."	attrs["description"] (line 303)
ResourceIDField	resource_id="user_id"	attrs["resource_id"] (line 304)
Internal	internal="true"	attrs["internal"] (line 309), parsed via parseBool
BootstrapSafe	bootstrap_safe="true"	attrs["bootstrap_safe"] (line 308), parsed via parseBool
FireAndForget	fire_and_forget="true"	attrs["fire_and_forget"] (line 310), parsed via parseBool
OutputAttr	output="model.LoginOutput"	attrs["output"] (lines 365-371), resolved against import map
From method signature (AST introspection):
Field	Source	How
Enabled	Always true	Hardcoded in render.go line 109
HasInput	*TIn parameter present	classifyParam() (line 412-413): a *ast.StarExpr param becomes "input" role
HasStream	<-chan dispatch.Item[TIn] parameter	classifyParam() (line 419-422): a *ast.ChanType (recv) with dispatch.Item[T] becomes "stream"
Input (TypeRef)	The TIn type	Resolved via typeRef() against the file's import map (lines 479-503)
Result	Return type shape	Analyzed in lines 338-363: (error) -> empty, (TOut, error) -> document, ([]TOut, error) -> documents, (*abstract.Result, error) -> raw
Output (TypeRef)	The TOut type	From the first return value via outputType() (line 348)
Service	Receiver type name	From receiverService() (lines 175-192)
MethodName	Function name	fd.Name.Name (line 305)
The classifyParam function (lines 409-439) resolves each parameter by AST type:
- context.Context -> ignored (role "ctx")
- abstract.Message -> ignored (role "msg")
- *ast.StarExpr -> input type (the *TIn convention)
- *ast.ChanType with dispatch.Item[T] -> stream type
- *ast.SelectorExpr -> cross-package selector (context/abstract)
4. The Template / Generation Logic for registrations.go
File: /home/augustine/projects/hestia/cmd/hestia/core/gen/render.go
There is no Go text/template -- the generation is entirely procedural string building using fmt.Fprintf into a bytes.Buffer.
renderRegistrations() (lines 15-100) builds the file:
1. Header: // Code generated by cmd/hestia service generate. DO NOT EDIT.
2. Package declaration
3. Import block -- collects needed imports:
- "context" only when a no-input handler needs an inline closure
- abstract and dispatch always
- Type-specific imports from input/output TypeRefs
4. RegisterService() function -- registers the service struct (<Pkg>Service) in the DI container
5. Registrations() function -- returns []abstract.MessageRegistration with one entry per annotation
writeRegistration() (lines 102-160) builds each registration entry:
For each Annotation, it emits:
- Name: from the annotation's message name
- Description: if non-empty
- Intent: mapped via verbConst() (e.g., VerbCreate -> abstract.Create)
- Enabled: true (always)
- BootstrapSafe: true if set
- Internal: true if set
- FireAndForget: true if set
- Input: block with Schema: dispatch.SchemaFromTypeWithTag[T]("input") and optional ResourceIDField, Streaming
- Output: via dispatch.SchemaFromType[T]() for document/documents results
- Handler: via handlerExpr() (lines 165-213)
handlerExpr() (lines 165-213) selects the dispatch adapter:
Condition	Emitted handler
Streaming input	dispatch.HandleInputStream[T](s.Method)
Raw result (*abstract.Result) with input	dispatch.Handle[T](s.Method)
Raw result, no input	inline closure wrapping s.Method(ctx, msg)
No input, empty result	inline closure: calls method, returns &abstract.Result{}
No input, document result	inline closure: calls method, calls .Document(), wraps with dispatch.NewDocumentResult
No input, documents result	inline closure: calls method, wraps with dispatch.NewDocumentsResult
With input, empty result	dispatch.HandleEmpty[T](s.Method)
With input, document result	dispatch.HandleDocument[TIn, *TOut](s.Method)
With input, documents result	dispatch.HandleDocuments[T](s.Method)
5. Annotation / Comment-Based Configuration
The annotation format is @hestia.register(...) placed in Go doc comments on methods. Multiple blocks are allowed on a single method.
Full attribute reference:
// @hestia.register(
//   name="system:module:resource:verb",     // REQUIRED - 4-segment message name
//   intent="create|read|update|delete|query|stream|check",  // REQUIRED
//   rule="administrator|authenticated|public",  // policy rule key (default: "authenticated")
//   description="Human-readable description",
//   resource_id="field_name",               // route identity field from input struct
//   bootstrap_safe="true",                  // handler works before DB is populated
//   internal="true",                        // not routed over HTTP (e.g. inter-service calls)
//   fire_and_forget="true",                 // transport responds immediately (HTTP 202)
//   output="model.TypeName",                // explicit output type when signature alone can't express it
// )
Concrete examples from auth/service.go:
With input, document result, bootstrap_safe:
// @hestia.register(
//   name="system:auth:session:create",
//   intent="create",
//   rule="public",
//   bootstrap_safe="true",
//   description="Authenticate and receive a session token",
//   output="model.LoginOutput",
// )
func (s *AuthService) CreateSession(ctx context.Context, msg abstract.Message, input *model.LoginInput) (*abstract.Result, error) {
Note: The method returns (*abstract.Result, error) which is ResultRaw, but the output="model.LoginOutput" attribute overrides the Output schema.
With input, empty result, bootstrap_safe:
// @hestia.register(
//   name="system:auth:session:delete",
//   intent="delete",
//   rule="authenticated",
//   bootstrap_safe="true",
//   description="Logout",
// )
func (s *AuthService) DeleteSession(ctx context.Context, msg abstract.Message, input *model.DeleteSessionInput) error {
No input, document result, internal:
// @hestia.register(
//   name="system:auth:session:validate",
//   intent="read",
//   rule="public",
//   internal="true",
//   description="Validate a session token",
//   output="model.ClaimsOutput",
// )
func (s *AuthService) ValidateSession(ctx context.Context, msg abstract.Message) (*model.ClaimsDocumentView, error) {
Streaming input (from test fixture):
// @hestia.register(
//   name="system:users:user:import",
//   intent="create",
//   rule="administrator",
//   description="Bulk-import users from an NDJSON stream",
// )
func (s *UsersService) ImportUsers(ctx context.Context, msg abstract.Message, items <-chan dispatch.Item[ImportUserInput]) (*abstract.Result, error) {
6. Post-Generation: Collector Regeneration
After generating per-service files, refreshCollector() (service.go lines 379-401) regenerates two module-level collectors:
1. services.go (gen.GenerateCollector) -- discovers all subdirectories of the module that contain registrations.go, then emits RegisterServices(rt) (DI registration) and CollectServiceRegistrations(rt) (message collection).
2. gen_sanitization.go (gen.GenerateSanitizationCollector) -- discovers subdirectories containing a SanitizationRules() function in sanitization.go, then emits a map from feature scope to *sanitize.FieldMaskConfig.
Summary of the Full Pipeline
hestia service generate system auth
  |
  v
serviceModuleDir("system") -> core/system
  |
  v
gen.Generate("core/system/auth")
  |
  +-- annotate.Parse("core/system/auth")
  |     |-- Parse all .go files (skip _test.go) with go/parser
  |     |-- For each FuncDecl with doc + receiver:
  |     |     receiverService() -> must end in "Service"
  |     |     annotationBlocks() -> extract @hestia.register(...) key="value" pairs
  |     |     buildAnnotation() -> introspect params + return types via AST
  |     +-- Return []Annotation
  |
  +-- renderRegistrations(pkg, anns)
  |     |-- Emit header, package, imports
  |     |-- Emit RegisterService() for DI
  |     +-- Emit Registrations() with one MessageRegistration per annotation
  |           Intent from verbConst(), Input from *TIn param, Output from return type
  |           Handler via dispatch.Handle*/HandleEmpty*/HandleDocument*/etc.
  |
  +-- renderPolicies(pkg, anns)
  |     +-- Emit Policies() returning []policies.Binding per annotation
  |
  +-- writeFormatted() -> go/format + os.WriteFile
  |
  v
gen.SeedSanitization("core/system/auth") -> write sanitization.go if missing
  |
  v
refreshCollector("core/system")
  |-- gen.GenerateCollector() -> services.go (DI + collector)
  +-- gen.GenerateSanitizationCollector() -> gen_sanitization.go
