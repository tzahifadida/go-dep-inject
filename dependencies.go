// Package godepinject provides a dependency injection system for managing and retrieving dependencies.
//
// This package allows for registering, initializing, and retrieving dependencies in a flexible
// and type-safe manner. It supports dependency namespaces, qualifiers, and deadlock detection.
// The library leverages Go's generics to provide type-safe dependency registration and retrieval,
// allowing for compile-time type checking and improved code safety.
//
// Configuration:
//   - Use context.WithValue(ctx, DependencyNamespaceKey, uuid.New()) to create a new namespace.
//   - Use SetDeadlockTimeout(ctx, duration) to set a custom deadlock detection timeout.
//
// Key Features:
//   - Generic-based dependency registration and retrieval for type safety.
//   - Support for dependency qualifiers to distinguish between multiple instances of the same type.
//   - Automatic dependency initialization with cycle detection.
//   - Namespace support for isolating dependencies in different parts of your application.
package godepinject

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

// DependencyNamespaceKey is the key used to store the namespace UUID in the context.
const DependencyNamespaceKey ContextKey = "DependencyNamespaceKey"

// DependencyDeadlockTimeoutKey is the key used to store the deadlock detection timeout in the context.
const DependencyDeadlockTimeoutKey ContextKey = "DependencyDeadlockTimeoutKey"

// DefaultDeadlockTimeout is the default timeout for deadlock detection if not specified.
const DefaultDeadlockTimeout = 30 * time.Second

// ErrDependencyNotFound is returned when a requested dependency is not found.
var ErrDependencyNotFound = fmt.Errorf("dependency not found")

// initializable is an interface that must be implemented by all dependencies.
type initializable interface {
	// Init initializes the dependency with the given context.
	Init(ctx context.Context)
}

// DependencyInitStage represents the initialization stage of a dependency.
type DependencyInitStage int

const (
	// DependencyInitStageNotInitialized indicates that the dependency has not been initialized.
	DependencyInitStageNotInitialized DependencyInitStage = iota
	// DependencyInitStageInitialized indicates that the dependency has been initialized.
	DependencyInitStageInitialized
)

// dependencyState represents the state of a registered dependency.
type dependencyState struct {
	initialized bool
	initStage   DependencyInitStage
	d           initializable
	qualifier   string
	initLock    chan struct{}
}

var dependenciesNamespace = make(map[string]map[reflect.Type][]*dependencyState)
var dependenciesNamespaceRWLock sync.RWMutex

// DependencyInitializationError represents an error that occurred during dependency initialization.
type DependencyInitializationError struct {
	Message string
	Stack   []string // Contains combined dependency and qualifier names
	Cause   error    // Holds the previous error in the chain
}

// Error returns a string representation of the DependencyInitializationError.
func (e *DependencyInitializationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s\nDependency stack: %v\nCaused by: %v", e.Message, e.Stack, e.Cause)
	}
	return fmt.Sprintf("%s\nDependency stack: %v", e.Message, e.Stack)
}

// Unwrap returns the underlying cause of the DependencyInitializationError.
func (e *DependencyInitializationError) Unwrap() error {
	return e.Cause
}

// initializationState tracks the state of dependency initialization.
type initializationState int

const (
	notVisited initializationState = iota
	inProgress
	completed
)

// initializationContext tracks the initialization process.
type initializationContext struct {
	mu      sync.Mutex
	stack   []string
	visited map[string]initializationState
}

// initializationContextKey is the context key for the initializationContext.
const initializationContextKey ContextKey = "InitializationContextKey"

// combineDependencyAndQualifier combines a dependency name and qualifier.
func combineDependencyAndQualifier(depName, qualifier string) string {
	if qualifier == "" {
		return depName
	}
	return fmt.Sprintf("%s (%s)", depName, qualifier)
}

// TypeOfInterface returns the reflect.Type of the given interface.
func TypeOfInterface[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// RegisterDependency registers a new dependency with an optional qualifier.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - dependency: The dependency to register, must implement the initializable interface.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies of the same type.
//
// Returns an error if the dependency could not be registered.
func RegisterDependency(ctx context.Context, dependency initializable, qualifier ...string) error {
	namespace := ctx.Value(DependencyNamespaceKey).(string)
	dependenciesNamespaceRWLock.Lock()
	defer dependenciesNamespaceRWLock.Unlock()

	q := ""
	if len(qualifier) > 0 {
		q = qualifier[0]
	}

	if _, ok := dependenciesNamespace[namespace]; !ok {
		dependenciesNamespace[namespace] = make(map[reflect.Type][]*dependencyState)
	}

	underlyingType := reflect.TypeOf(dependency)
	newState := &dependencyState{
		initialized: false,
		initStage:   DependencyInitStageNotInitialized,
		d:           dependency,
		qualifier:   q,
		initLock:    make(chan struct{}, 1), // Create a buffered channel for locking
	}

	if dependencies, ok := dependenciesNamespace[namespace][underlyingType]; ok {
		for _, dep := range dependencies {
			if dep.qualifier == q {
				return fmt.Errorf("dependency with qualifier '%s' already registered for type %v", q, underlyingType)
			}
		}
		dependenciesNamespace[namespace][underlyingType] = append(dependencies, newState)
	} else {
		dependenciesNamespace[namespace][underlyingType] = []*dependencyState{newState}
	}

	return nil
}

// MustRegisterDependency is like RegisterDependency but panics if an error occurs.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - dependency: The dependency to register, must implement the initializable interface.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies of the same type.
func MustRegisterDependency(ctx context.Context, dependency initializable, qualifier ...string) {
	if err := RegisterDependency(ctx, dependency, qualifier...); err != nil {
		panic(err)
	}
}

// getDeadlockTimeout retrieves the deadlock detection timeout from the context or returns the default.
func getDeadlockTimeout(ctx context.Context) time.Duration {
	if timeout, ok := ctx.Value(DependencyDeadlockTimeoutKey).(time.Duration); ok {
		return timeout
	}
	return DefaultDeadlockTimeout
}

// getDependencyID returns a unique identifier for a dependency.
func getDependencyID(dep initializable, qualifier string) string {
	return fmt.Sprintf("%T:%s", dep, qualifier)
}

// initializeAndReturn initializes a dependency and returns it.
func initializeAndReturn(ctx context.Context, dep *dependencyState) (any, error) {
	// Get or create the initializationContext
	var initCtx *initializationContext
	if ctx.Value(initializationContextKey) == nil {
		initCtx = &initializationContext{
			visited: make(map[string]initializationState),
			stack:   []string{},
		}
		ctx = context.WithValue(ctx, initializationContextKey, initCtx)
	} else {
		initCtx = ctx.Value(initializationContextKey).(*initializationContext)
	}

	// Check for cyclic dependencies before attempting to acquire the lock
	depID := getDependencyID(dep.d, dep.qualifier)

	initCtx.mu.Lock()
	if state, exists := initCtx.visited[depID]; exists {
		if state == inProgress {
			stack := make([]string, len(initCtx.stack)+1)
			copy(stack, initCtx.stack)
			stack[len(stack)-1] = depID
			initCtx.mu.Unlock()
			return nil, &DependencyInitializationError{
				Message: "Circular dependency detected during initialization",
				Stack:   stack,
			}
		}
		if state == completed {
			initCtx.mu.Unlock()
			return dep.d, nil // Already initialized
		}
	}
	initCtx.visited[depID] = inProgress
	initCtx.stack = append(initCtx.stack, depID)
	initCtx.mu.Unlock()

	// Now attempt to acquire the lock with a timeout
	timeout := getDeadlockTimeout(ctx)
	select {
	case dep.initLock <- struct{}{}:
		// Lock acquired, defer its release
		defer func() { <-dep.initLock }()
	case <-time.After(timeout):
		return nil, &DependencyInitializationError{
			Message: "Timeout while waiting to initialize dependency",
			Stack:   []string{depID},
		}
	}

	// Initialize the dependency
	err := initializeDependency(ctx, dep)
	if err != nil {
		return nil, err
	}

	return dep.d, nil
}

// initializeDependency initializes a dependency.
func initializeDependency(ctx context.Context, dep *dependencyState) (err error) {
	// Defer panic recovery
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case *DependencyInitializationError:
				err = e
			case error:
				// If it's another type of error, wrap it in our error type
				err = &DependencyInitializationError{
					Message: fmt.Sprintf("Panic during dependency initialization: %v", e),
					Stack:   []string{getDependencyID(dep.d, dep.qualifier)},
					Cause:   e,
				}
			default:
				// If it's not an error, create a new error
				err = &DependencyInitializationError{
					Message: fmt.Sprintf("Panic during dependency initialization: %v", r),
					Stack:   []string{getDependencyID(dep.d, dep.qualifier)},
				}
			}
		}
	}()

	// Initialize the dependency
	dep.d.Init(ctx)

	// Mark as completed
	initCtx := ctx.Value(initializationContextKey).(*initializationContext)
	depID := getDependencyID(dep.d, dep.qualifier)
	initCtx.mu.Lock()
	initCtx.visited[depID] = completed
	initCtx.stack = initCtx.stack[:len(initCtx.stack)-1]
	initCtx.mu.Unlock()

	dep.initStage = DependencyInitStageInitialized
	return nil
}

// GetDependency retrieves a dependency of the specified type and optional qualifier.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - dependency: A zero value of the dependency type to retrieve.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies of the same type.
//
// Returns the initialized dependency and an error if it could not be retrieved or initialized.
func GetDependency(ctx context.Context, dependency initializable, qualifier ...string) (any, error) {
	namespace := ctx.Value(DependencyNamespaceKey).(string)
	dependenciesNamespaceRWLock.RLock()
	defer dependenciesNamespaceRWLock.RUnlock()

	q := ""
	if len(qualifier) > 0 {
		q = qualifier[0]
	}

	if dependencies, ok := dependenciesNamespace[namespace]; ok {
		underlyingType := reflect.TypeOf(dependency)
		if deps, ok := dependencies[underlyingType]; ok {
			for _, dep := range deps {
				if q == "" || dep.qualifier == q {
					dependenciesNamespaceRWLock.RUnlock()
					result, err := initializeAndReturn(ctx, dep)
					dependenciesNamespaceRWLock.RLock()
					return result, err
				}
			}
		}
	}

	return nil, ErrDependencyNotFound
}

// MustGetDependency is like GetDependency but panics if an error occurs.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - dependency: A zero value of the dependency type to retrieve.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies of the same type.
//
// Returns the initialized dependency.
func MustGetDependency(ctx context.Context, dependency initializable, qualifier ...string) any {
	dep, err := GetDependency(ctx, dependency, qualifier...)
	if err != nil {
		panic(err)
	}
	return dep
}

// GetDependencyByInterface retrieves a dependency that implements the specified interface.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - interfaceType: The reflect.Type of the interface to match.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies implementing the same interface.
//
// Returns the initialized dependency and an error if it could not be retrieved or initialized.
func GetDependencyByInterface(ctx context.Context, interfaceType reflect.Type, qualifier ...string) (any, error) {
	namespace := ctx.Value(DependencyNamespaceKey).(string)
	dependenciesNamespaceRWLock.RLock()
	defer dependenciesNamespaceRWLock.RUnlock()

	q := ""
	if len(qualifier) > 0 {
		q = qualifier[0]
	}

	if dependencies, ok := dependenciesNamespace[namespace]; ok {
		var foundDependencies []any

		for _, deps := range dependencies {
			for _, dep := range deps {
				if reflect.TypeOf(dep.d).Implements(interfaceType) && (q == "" || dep.qualifier == q) {
					dependenciesNamespaceRWLock.RUnlock()
					initializedDep, err := initializeAndReturn(ctx, dep)
					dependenciesNamespaceRWLock.RLock()
					if err != nil {
						return nil, err
					}
					foundDependencies = append(foundDependencies, initializedDep)
				}
			}
		}

		if len(foundDependencies) == 1 {
			return foundDependencies[0], nil
		} else if len(foundDependencies) > 1 && q == "" {
			return nil, &DependencyInitializationError{
				Message: "Multiple dependencies implement the given interface, please provide a qualifier",
				Stack:   []string{combineDependencyAndQualifier(interfaceType.String(), q)},
			}
		}
	}

	return nil, ErrDependencyNotFound
}

// MustGetDependencyByInterface is like GetDependencyByInterface but panics if an error occurs.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - interfaceType: The reflect.Type of the interface to match.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies implementing the same interface.
//
// Returns the initialized dependency.
func MustGetDependencyByInterface(ctx context.Context, interfaceType reflect.Type, qualifier ...string) any {
	dep, err := GetDependencyByInterface(ctx, interfaceType, qualifier...)
	if err != nil {
		panic(err)
	}
	return dep
}

// GetDependencyT is a generic version of GetDependency that uses Go's generics to provide type-safe dependency retrieval.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies of the same type.
//
// Returns the initialized dependency of type T and an error if it could not be retrieved or initialized.
// This function leverages Go's generics to ensure type safety at compile-time.
func GetDependencyT[T any](ctx context.Context, qualifier ...string) (T, error) {
	namespace := ctx.Value(DependencyNamespaceKey).(string)
	dependenciesNamespaceRWLock.RLock()
	defer dependenciesNamespaceRWLock.RUnlock()

	var zero T
	q := ""
	if len(qualifier) > 0 {
		q = qualifier[0]
	}

	if dependencies, ok := dependenciesNamespace[namespace]; ok {
		underlyingType := reflect.TypeOf((*T)(nil)).Elem()
		if deps, ok := dependencies[underlyingType]; ok {
			for _, dep := range deps {
				if q == "" || dep.qualifier == q {
					dependenciesNamespaceRWLock.RUnlock()
					initializedDep, err := initializeAndReturn(ctx, dep)
					dependenciesNamespaceRWLock.RLock()
					if err != nil {
						return zero, err
					}
					return initializedDep.(T), nil
				}
			}
		}
	}

	return zero, ErrDependencyNotFound
}

// MustGetDependencyT is like GetDependencyT but panics if an error occurs. It uses generics for type-safe dependency retrieval.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies of the same type.
//
// Returns the initialized dependency of type T.
// This function leverages Go's generics to ensure type safety at compile-time.
func MustGetDependencyT[T any](ctx context.Context, qualifier ...string) T {
	dep, err := GetDependencyT[T](ctx, qualifier...)
	if err != nil {
		panic(err)
	}
	return dep
}

// GetDependencyByInterfaceType is a generic version of GetDependencyByInterface that uses Go's generics for type-safe interface-based dependency retrieval.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies implementing the same interface.
//
// Returns the initialized dependency implementing interface I and an error if it could not be retrieved or initialized.
// This function leverages Go's generics to ensure type safety at compile-time when working with interfaces.
func GetDependencyByInterfaceType[I any](ctx context.Context, qualifier ...string) (I, error) {
	var zero I
	interfaceType := reflect.TypeOf((*I)(nil)).Elem()

	dep, err := GetDependencyByInterface(ctx, interfaceType, qualifier...)
	if err != nil {
		return zero, err
	}

	return dep.(I), nil
}

// MustGetDependencyByInterfaceType is like GetDependencyByInterfaceType but panics if an error occurs. It uses generics for type-safe interface-based dependency retrieval.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - qualifier: Optional qualifier to distinguish between multiple dependencies implementing the same interface.
//
// Returns the initialized dependency implementing interface I.
// This function leverages Go's generics to ensure type safety at compile-time when working with interfaces.
func MustGetDependencyByInterfaceType[I any](ctx context.Context, qualifier ...string) I {
	dep, err := GetDependencyByInterfaceType[I](ctx, qualifier...)
	if err != nil {
		panic(err)
	}
	return dep
}

// GetDependencyByQualifier retrieves a dependency by its qualifier.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - qualifier: The qualifier of the dependency to retrieve.
//
// Returns the initialized dependency and an error if it could not be retrieved or initialized.
func GetDependencyByQualifier(ctx context.Context, qualifier string) (any, error) {
	namespace := ctx.Value(DependencyNamespaceKey).(string)
	dependenciesNamespaceRWLock.RLock()
	defer dependenciesNamespaceRWLock.RUnlock()

	if dependencies, ok := dependenciesNamespace[namespace]; ok {
		for _, deps := range dependencies {
			for _, dep := range deps {
				if dep.qualifier == qualifier {
					dependenciesNamespaceRWLock.RUnlock()
					result, err := initializeAndReturn(ctx, dep)
					dependenciesNamespaceRWLock.RLock()
					return result, err
				}
			}
		}
	}

	return nil, ErrDependencyNotFound
}

// MustGetDependencyByQualifier is like GetDependencyByQualifier but panics if an error occurs.
//
// Parameters:
//   - ctx: The context containing the dependency namespace.
//   - qualifier: The qualifier of the dependency to retrieve.
//
// Returns the initialized dependency.
func MustGetDependencyByQualifier(ctx context.Context, qualifier string) any {
	dep, err := GetDependencyByQualifier(ctx, qualifier)
	if err != nil {
		panic(err)
	}
	return dep
}

// SetDeadlockTimeout sets a custom deadlock detection timeout for the given context.
//
// Parameters:
//   - ctx: The context to modify.
//   - timeout: The duration to set as the deadlock detection timeout.
//
// Returns a new context with the updated deadlock detection timeout.
func SetDeadlockTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, DependencyDeadlockTimeoutKey, timeout)
}
