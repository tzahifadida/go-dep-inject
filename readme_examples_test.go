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
	db := &BasicDatabase{connection: "Connected to PostgreSQL"}
	err := godepinject.RegisterDependency(ctx, db)
	if err != nil {
		t.Fatalf("Failed to register BasicDatabase: %v", err)
	}

	userService := &BasicUserService{}
	err = godepinject.RegisterDependency(ctx, userService)
	if err != nil {
		t.Fatalf("Failed to register BasicUserService: %v", err)
	}

	// Retrieve and use dependencies
	retrievedDB, err := godepinject.GetDependencyT[*BasicDatabase](ctx)
	if err != nil {
		t.Fatalf("Failed to get BasicDatabase: %v", err)
	}

	retrievedUserService, err := godepinject.GetDependencyT[*BasicUserService](ctx)
	if err != nil {
		t.Fatalf("Failed to get BasicUserService: %v", err)
	}
	retrievedUserService.db = retrievedDB

	result := retrievedUserService.GetUser()
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
	customLogger := &SimpleLogger{}

	// Register dependencies
	err := godepinject.RegisterDependency(ctx, customLogger)
	if err != nil {
		t.Fatalf("Failed to register SimpleLogger: %v", err)
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

// TestQualifierExample tests the qualifier example from the README
func TestQualifierExample(t *testing.T) {
	// Create a namespace
	namespace := uuid.New().String()
	ctx := context.WithValue(context.Background(), godepinject.DependencyNamespaceKey, namespace)

	// Register databases with qualifiers
	postgresDB := &PostgresDB{connectionString: "postgres://localhost:5432/mydb"}
	err := godepinject.RegisterDependency(ctx, postgresDB, "primary")
	if err != nil {
		t.Fatalf("Failed to register PostgresDB: %v", err)
	}

	mongoDB := &MongoDB{connectionString: "mongodb://localhost:27017/mydb"}
	err = godepinject.RegisterDependency(ctx, mongoDB, "secondary")
	if err != nil {
		t.Fatalf("Failed to register MongoDB: %v", err)
	}

	// Register QualifiedUserService
	err = godepinject.RegisterInitializableDependency(ctx, &QualifiedUserService{})
	if err != nil {
		t.Fatalf("Failed to register QualifiedUserService: %v", err)
	}

	// Use QualifiedUserService with primary (Postgres) database
	userService, err := godepinject.GetDependencyT[*QualifiedUserService](ctx)
	if err != nil {
		t.Fatalf("Failed to get QualifiedUserService: %v", err)
	}
	primaryResult := userService.GetUserData()
	expectedPrimary := "User data from Postgres: postgres://localhost:5432/mydb"
	if primaryResult != expectedPrimary {
		t.Errorf("Expected %q, but got %q", expectedPrimary, primaryResult)
	}

	// Create a new QualifiedUserService with MongoDB
	mongoUserService := &QualifiedUserService{}
	mongoCtx := context.WithValue(ctx, QualifierKey, "secondary")
	err = godepinject.RegisterInitializableDependency(mongoCtx, mongoUserService, "mongo")
	if err != nil {
		t.Fatalf("Failed to register mongo QualifiedUserService: %v", err)
	}

	// Use QualifiedUserService with secondary (MongoDB) database
	mongoUserService, err = godepinject.GetDependencyT[*QualifiedUserService](mongoCtx, "mongo")
	if err != nil {
		t.Fatalf("Failed to get mongo QualifiedUserService: %v", err)
	}
	secondaryResult := mongoUserService.GetUserData()
	expectedSecondary := "User data from MongoDB: mongodb://localhost:27017/mydb"
	if secondaryResult != expectedSecondary {
		t.Errorf("Expected %q, but got %q", expectedSecondary, secondaryResult)
	}

	// Directly use databases
	primaryDB, err := godepinject.GetDependencyByInterfaceType[QualifiedDatabase](ctx, "primary")
	if err != nil {
		t.Fatalf("Failed to get primary database: %v", err)
	}
	primaryDirectResult := primaryDB.GetConnection()
	if primaryDirectResult != "Postgres: postgres://localhost:5432/mydb" {
		t.Errorf("Expected %q, but got %q", "Postgres: postgres://localhost:5432/mydb", primaryDirectResult)
	}

	secondaryDB, err := godepinject.GetDependencyByInterfaceType[QualifiedDatabase](ctx, "secondary")
	if err != nil {
		t.Fatalf("Failed to get secondary database: %v", err)
	}
	secondaryDirectResult := secondaryDB.GetConnection()
	if secondaryDirectResult != "MongoDB: mongodb://localhost:27017/mydb" {
		t.Errorf("Expected %q, but got %q", "MongoDB: mongodb://localhost:27017/mydb", secondaryDirectResult)
	}
}

// Basic Example types
type BasicDatabase struct {
	connection string
}

func (d *BasicDatabase) GetConnection() string {
	return d.connection
}

type BasicUserService struct {
	db *BasicDatabase
}

func (u *BasicUserService) GetUser() string {
	return "User from " + u.db.GetConnection()
}

// Advanced Example types
type Logger interface {
	Log(message string)
}

type SimpleLogger struct {
	output strings.Builder
}

func (l *SimpleLogger) Log(message string) {
	l.output.WriteString("Log: " + message + "\n")
}

func (l *SimpleLogger) Output() string {
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

// Qualifier Example types
type QualifiedDatabase interface {
	GetConnection() string
}

type PostgresDB struct {
	connectionString string
}

func (p *PostgresDB) GetConnection() string {
	return "Postgres: " + p.connectionString
}

type MongoDB struct {
	connectionString string
}

func (m *MongoDB) GetConnection() string {
	return "MongoDB: " + m.connectionString
}

// QualifierKey is used to store the database qualifier in the context
const QualifierKey = "DatabaseQualifier"

type QualifiedUserService struct {
	db QualifiedDatabase
}

func (u *QualifiedUserService) Init(ctx context.Context) {
	var err error
	qualifier := "primary" // Default to primary if not specified
	if q, ok := ctx.Value(QualifierKey).(string); ok {
		qualifier = q
	}
	u.db, err = godepinject.GetDependencyByInterfaceType[QualifiedDatabase](ctx, qualifier)
	if err != nil {
		panic(err)
	}
}

func (u *QualifiedUserService) GetUserData() string {
	return "User data from " + u.db.GetConnection()
}
