package bloblang

import (
	"github.com/Jeffail/benthos/v3/internal/bloblang/parser"
	"github.com/Jeffail/benthos/v3/internal/bloblang/query"
)

// Environment provides an isolated Bloblang environment where the available
// features, functions and methods can be modified.
type Environment struct {
	functions *query.FunctionSet
	methods   *query.MethodSet
}

// NewEnvironment creates a fresh Bloblang environment, starting with the full
// range of globally defined features (functions and methods), and provides APIs
// for expanding or contracting the features available to this environment.
//
// It's worth using an environment when you need to restrict the access or
// capabilities that certain bloblang mappings have versus others.
//
// For example, an environment could be created that removes any functions for
// accessing environment variables or reading data from the host disk, which
// could be used in certain situations without removing those functions globally
// for all mappings.
func NewEnvironment() *Environment {
	return &Environment{
		functions: query.AllFunctions.Without(),
		methods:   query.AllMethods.Without(),
	}
}

// NewEmptyEnvironment creates a fresh Bloblang environment starting completely
// empty, where no functions or methods are initially available.
func NewEmptyEnvironment() *Environment {
	return &Environment{
		functions: query.NewFunctionSet(),
		methods:   query.NewMethodSet(),
	}
}

// Parse a Bloblang mapping using the Environment to determine the features
// (functions and methods) available to the mapping.
//
// When a parsing error occurs the error will be the type *ParseError, which
// gives access to the line and column where the error occurred, as well as a
// method for creating a well formatted error message.
func (e *Environment) Parse(blobl string) (*Executor, error) {
	pCtx := parser.GlobalContext()
	if e != nil {
		pCtx.Functions = e.functions
		pCtx.Methods = e.methods
	}
	exec, err := parser.ParseMapping(pCtx, "", blobl)
	if err != nil {
		return nil, internalToPublicParserError([]rune(blobl), err)
	}
	return newExecutor(exec), nil
}

// RegisterMethod adds a new Bloblang method to the environment. All method
// names must match the regular expression /^[a-z0-9]+(_[a-z0-9]+)*$/ (snake
// case).
func (e *Environment) RegisterMethod(name string, ctor MethodConstructor) error {
	return e.methods.Add(
		query.NewMethodSpec(name, "").InCategory(query.MethodCategoryPlugin, ""),
		func(target query.Function, args ...interface{}) (query.Function, error) {
			fn, err := ctor(args...)
			if err != nil {
				return nil, err
			}
			return query.ClosureFunction("method "+name, func(ctx query.FunctionContext) (interface{}, error) {
				v, err := target.Exec(ctx)
				if err != nil {
					return nil, err
				}
				return fn(v)
			}, target.QueryTargets), nil
		},
		true,
	)
}

// WithoutMethods returns a copy of the environment but with a variadic list of
// method names removed. Instantiation of these removed methods within a mapping
// will cause errors at parse time.
func (e *Environment) WithoutMethods(names ...string) *Environment {
	return &Environment{
		functions: e.functions,
		methods:   e.methods.Without(names...),
	}
}

// WithoutFunctions returns a copy of the environment but with a variadic list
// of function names removed. Instantiation of these removed functions within a
// mapping will cause errors at parse time.
func (e *Environment) WithoutFunctions(names ...string) *Environment {
	return &Environment{
		functions: e.functions.Without(names...),
		methods:   e.methods,
	}
}

// RegisterFunction adds a new Bloblang function to the environment. All
// function names must match the regular expression /^[a-z0-9]+(_[a-z0-9]+)*$/
// (snake case).
func (e *Environment) RegisterFunction(name string, ctor FunctionConstructor) error {
	return e.functions.Add(
		query.NewFunctionSpec(query.FunctionCategoryPlugin, name, ""),
		func(args ...interface{}) (query.Function, error) {
			fn, err := ctor(args...)
			if err != nil {
				return nil, err
			}
			return query.ClosureFunction("function "+name, func(ctx query.FunctionContext) (interface{}, error) {
				return fn()
			}, nil), nil
		},
		true,
	)
}

//------------------------------------------------------------------------------

// Parse a Bloblang mapping allowing the use of the globally accessible range of
// features (functions and methods).
//
// When a parsing error occurs the error will be the type *ParseError, which
// gives access to the line and column where the error occurred, as well as a
// method for creating a well formatted error message.
func Parse(blobl string) (*Executor, error) {
	exec, err := parser.ParseMapping(parser.GlobalContext(), "", blobl)
	if err != nil {
		return nil, internalToPublicParserError([]rune(blobl), err)
	}
	return newExecutor(exec), nil
}

// RegisterMethod adds a new Bloblang method to the global enviromment. All
// method names must match the regular expression /^[a-z0-9]+(_[a-z0-9]+)*$/
// (snake case).
func RegisterMethod(name string, ctor MethodConstructor) error {
	return query.AllMethods.Add(
		query.NewMethodSpec(name, "").InCategory(query.MethodCategoryPlugin, ""),
		func(target query.Function, args ...interface{}) (query.Function, error) {
			fn, err := ctor(args...)
			if err != nil {
				return nil, err
			}
			return query.ClosureFunction("method "+name, func(ctx query.FunctionContext) (interface{}, error) {
				v, err := target.Exec(ctx)
				if err != nil {
					return nil, err
				}
				return fn(v)
			}, target.QueryTargets), nil
		},
		true,
	)
}

// RegisterFunction adds a new Bloblang function to the global enviromment. All
// function names must match the regular expression /^[a-z0-9]+(_[a-z0-9]+)*$/
// (snake case).
func RegisterFunction(name string, ctor FunctionConstructor) error {
	return query.AllFunctions.Add(
		query.NewFunctionSpec(query.FunctionCategoryPlugin, name, ""),
		func(args ...interface{}) (query.Function, error) {
			fn, err := ctor(args...)
			if err != nil {
				return nil, err
			}
			return query.ClosureFunction("function "+name, func(ctx query.FunctionContext) (interface{}, error) {
				return fn()
			}, nil), nil
		},
		true,
	)
}
