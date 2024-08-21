# go-dep-inject

⭐ If you find this project useful, please consider giving it a star! Your support helps us grow and improve. ⭐

go-dep-inject is a powerful and flexible dependency injection library for Go, leveraging generics for type-safe dependency management.

## Table of Contents

- [Installation](#installation)
- [Features](#features)
- [Usage](#usage)
- [Working with Namespaces](#working-with-namespaces)
- [Basic Example](#basic-example)
- [Advanced Example](#advanced-example)
- [Thread Safety](#thread-safety)
- [License](#license)
- [Contributing](#contributing)
- [Support](#support)

## Installation

To install go-dep-inject, use `go get`:

```bash
go get github.com/tzahifadida/go-dep-inject
```

## Features

- Generic-based dependency registration and retrieval for type safety
- "Thread-safe" and concurrent-safe operations
- Support for dependency qualifiers to distinguish between multiple instances of the same type
- Automatic dependency initialization with cycle detection
- Namespace support for isolating dependencies in different parts of your application
- Deadlock detection and configurable timeout
- Panic recovery during dependency initialization

## Usage

1. Import the package:
   ```go
   import "github.com/tzahifadida/go-dep-inject"
   ```

2. Create a namespace using a context:
   ```go
   ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, uuid.New().String())
   ```

3. Register dependencies:
   ```go
   godepinject.RegisterDependency(ctx, &MyDependency{})
   ```

4. Retrieve dependencies:
   ```go
   // For interfaces:
   dep, err := godepinject.GetDependencyByInterfaceType[MyInterface](ctx)
   
   // For concrete types:
   dep, err := godepinject.GetDependencyT[*MyConcreteType](ctx)
   ```

Note: `GetDependencyByInterfaceType` searches for and returns a dependency that implements the specified interface. This allows for loose coupling between your components, as you can work with interfaces rather than concrete types. Use `GetDependencyT` for retrieving concrete types.

Important: When you retrieve a dependency for the first time, its `Init` function is automatically called once. Subsequent retrievals of the same dependency will return the initialized instance without calling `Init` again. This ensures that each dependency is properly initialized exactly once, regardless of how many times it's retrieved or injected into other dependencies.

## Working with Namespaces

Namespaces are a crucial feature of go-dep-inject, allowing you to isolate dependencies in different parts of your application. Here's how to effectively use namespaces:

1. Create a unique namespace for each independent part of your application:
   ```go
   namespace := uuid.New().String()
   ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, namespace)
   ```

2. Always use the context with the namespace when registering or retrieving dependencies:
   ```go
   godepinject.RegisterDependency(ctx, &MyDependency{})
   dep, err := godepinject.GetDependencyT[*MyDependency](ctx)
   ```

3. Pass the context with the namespace to any function or method that needs to access dependencies:
   ```go
   func InitializeService(ctx context.Context) {
       service, err := godepinject.GetDependencyByInterfaceType[MyService](ctx)
       // ...
   }
   ```

4. For long-running applications, you can store the namespace string and recreate the context when needed:
   ```go
   type App struct {
       namespace string
   }

   func (a *App) GetContext() context.Context {
       return context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, a.namespace)
   }

   func (a *App) GetDependency() {
       ctx := a.GetContext()
       dep, err := godepinject.GetDependencyT[*MyDependency](ctx)
       // ...
   }
   ```

Remember that dependencies registered in one namespace are not available in another. This isolation allows you to have multiple instances of the same dependency type in different parts of your application without conflicts.

## Basic Example

Here's a basic example demonstrating how to use go-dep-inject with both interface-based and concrete type dependency retrieval:

```go
package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tzahifadida/go-dep-inject"
)

type Database interface {
	GetConnection() string
}

type PostgresDB struct {
	connection string
}

func (d *PostgresDB) Init(ctx context.Context) {
	d.connection = "Connected to PostgreSQL"
}

func (d *PostgresDB) GetConnection() string {
	return d.connection
}

type UserService struct {
	db Database
}

func (u *UserService) Init(ctx context.Context) {
	var err error
	// Retrieves a dependency that implements the Database interface
	u.db, err = godepinject.GetDependencyByInterfaceType[Database](ctx)
	if err != nil {
		panic(err)
	}
}

func (u *UserService) GetUser() string {
	return "User from " + u.db.GetConnection()
}

func main() {
	// Create a namespace
	namespace := uuid.New().String()
	ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, namespace)

	// Register dependencies
	godepinject.MustRegisterDependency(ctx, &PostgresDB{})
	godepinject.MustRegisterDependency(ctx, &UserService{})

	// Retrieve and use a dependency
	// Use GetDependencyT for concrete types like UserService
	userService, err := godepinject.GetDependencyT[*UserService](ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(userService.GetUser())
}
```

This example demonstrates:
1. Creating a namespace using a context
2. Defining and implementing an interface (Database)
3. Registering a concrete implementation (PostgresDB) and a service (UserService)
4. Retrieving dependencies by interface (Database in UserService.Init)
5. Retrieving concrete type dependencies (UserService in main)
6. Using the injected dependencies

## Advanced Example

Here's an advanced example showing dependencies injecting other dependencies and retrieving them by interface:

```go
package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tzahifadida/go-dep-inject"
)

type Logger interface {
	Log(message string)
}

type SimpleLogger struct{}

func (l *SimpleLogger) Init(ctx context.Context) {}

func (l *SimpleLogger) Log(message string) {
	fmt.Println("Log:", message)
}

type DataStore interface {
	Store(data string)
}

type MemoryStore struct {
	logger Logger
	data   []string
}

func (m *MemoryStore) Init(ctx context.Context) {
	var err error
	m.logger, err = godepinject.GetDependencyByInterfaceType[Logger](ctx)
	if err != nil {
		panic(err)
	}
	m.logger.Log("MemoryStore initialized")
}

func (m *MemoryStore) Store(data string) {
	m.data = append(m.data, data)
	m.logger.Log("Data stored: " + data)
}

type AppService struct {
	store  DataStore
	logger Logger
}

func (a *AppService) Init(ctx context.Context) {
	var err error
	a.store, err = godepinject.GetDependencyByInterfaceType[DataStore](ctx)
	if err != nil {
		panic(err)
	}
	a.logger, err = godepinject.GetDependencyByInterfaceType[Logger](ctx)
	if err != nil {
		panic(err)
	}
	a.logger.Log("AppService initialized")
}

func (a *AppService) RunApp() {
	a.store.Store("Important data")
	a.logger.Log("App finished running")
}

func main() {
	// Create a namespace
	namespace := uuid.New().String()
	ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, namespace)

	// Register dependencies
	godepinject.MustRegisterDependency(ctx, &SimpleLogger{})
	godepinject.MustRegisterDependency(ctx, &MemoryStore{})
	godepinject.MustRegisterDependency(ctx, &AppService{})

	// Retrieve and use the AppService
	appService, err := godepinject.GetDependencyT[*AppService](ctx)
	if err != nil {
		panic(err)
	}
	appService.RunApp()
}
```

This advanced example demonstrates:
1. Multiple interfaces and implementations (Logger, DataStore)
2. Dependencies injecting other dependencies (MemoryStore depends on Logger)
3. A service using multiple injected dependencies (AppService uses both DataStore and Logger)
4. Registering concrete implementations for interfaces
5. Retrieving dependencies by interface within Init methods
6. Retrieving a concrete type (AppService) in the main function

Note: The Init method of each dependency is automatically called when the dependency is first retrieved. This ensures that all required dependencies are properly initialized before use.

## Thread Safety

go-dep-inject is designed to be "thread-safe" and can be safely used in concurrent environments. All operations on dependencies, including registration, retrieval, and initialization, are protected by appropriate synchronization mechanisms. This ensures that you can use the library in multi-goroutine applications without worrying about race conditions or data corruption.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

If you encounter any problems or have any questions, please open an issue on the [GitHub repository](https://github.com/tzahifadida/go-dep-inject/issues).