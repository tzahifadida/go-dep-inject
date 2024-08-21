package godepinject_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/tzahifadida/go-dep-inject"
	"strings"
	"testing"
)

// TestBasicExample tests the basic example from the README
func TestBasicExample(t *testing.T) {
	// Create a namespace
	namespace := uuid.New().String()
	ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, namespace)

	// Register dependencies
	err := godepinject.RegisterInitializableDependency(ctx, &PostgresDB{})
	if err != nil {
		t.Fatalf("Failed to register PostgresDB: %v", err)
	}
	err = godepinject.RegisterInitializableDependency(ctx, &UserService{})
	if err != nil {
		t.Fatalf("Failed to register UserService: %v", err)
	}

	// Retrieve and use a dependency
	userService, err := godepinject.GetDependencyT[*UserService](ctx)
	if err != nil {
		t.Fatalf("Failed to get UserService: %v", err)
	}

	result := userService.GetUser()
	expected := "User from Connected to PostgreSQL"
	if result != expected {
		t.Errorf("Expected %q, but got %q", expected, result)
	}
}

// TestAdvancedExample tests the advanced example from the README
func TestAdvancedExample(t *testing.T) {
	// Create a namespace
	namespace := uuid.New().String()
	ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, namespace)

	// Create a custom logger to capture output
	customLogger := &CustomLogger{}

	// Register dependencies
	err := godepinject.RegisterInitializableDependency(ctx, customLogger)
	if err != nil {
		t.Fatalf("Failed to register CustomLogger: %v", err)
	}
	err = godepinject.RegisterInitializableDependency(ctx, &MemoryStore{})
	if err != nil {
		t.Fatalf("Failed to register MemoryStore: %v", err)
	}
	err = godepinject.RegisterInitializableDependency(ctx, &AppService{})
	if err != nil {
		t.Fatalf("Failed to register AppService: %v", err)
	}

	// Retrieve and use the AppService
	appService, err := godepinject.GetDependencyT[*AppService](ctx)
	if err != nil {
		t.Fatalf("Failed to get AppService: %v", err)
	}

	appService.RunApp()

	expectedOutput := `Log: MemoryStore initialized
Log: AppService initialized
Log: Data stored: Important data
Log: App finished running
`
	if customLogger.Output() != expectedOutput {
		t.Errorf("Expected output:\n%s\nBut got:\n%s", expectedOutput, customLogger.Output())
	}
}

// Below are the type definitions and implementations used in the examples

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
	u.db, err = godepinject.GetDependencyByInterfaceType[Database](ctx)
	if err != nil {
		panic(err)
	}
}

func (u *UserService) GetUser() string {
	return "User from " + u.db.GetConnection()
}

type Logger interface {
	Log(message string)
}

// CustomLogger is used to capture log output for testing
type CustomLogger struct {
	output strings.Builder
}

func (l *CustomLogger) Init(ctx context.Context) {}

func (l *CustomLogger) Log(message string) {
	l.output.WriteString("Log: " + message + "\n")
}

func (l *CustomLogger) Output() string {
	return l.output.String()
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
