package godepinject

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type testDependency struct {
	initialized bool
}

func (t *testDependency) Init(ctx context.Context) {
	t.initialized = true
}

type testInterface interface {
	TestMethod()
}

type testImplementation struct {
	testDependency
}

func (t *testImplementation) TestMethod() {}

func TestRegisterAndGetDependency(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	err := RegisterInitializableDependency(ctx, dep, "test")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	retrieved, err := GetDependency(ctx, dep, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency: %v", err)
	}

	if retrieved != dep {
		t.Errorf("Retrieved dependency is not the same as registered")
	}

	if !retrieved.(*testDependency).initialized {
		t.Errorf("Retrieved dependency was not initialized")
	}
}

func TestGetNonExistentDependency(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	_, err := GetDependency(ctx, dep, "nonexistent")
	if err == nil {
		t.Errorf("Expected error when getting non-existent dependency, got nil")
	}

	if !errors.Is(err, ErrDependencyNotFound) {
		t.Errorf("Expected DependencyInitializationError, got %T", err)
	}
}

func TestRegisterDuplicateDependency(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep1 := &testDependency{}
	dep2 := &testDependency{}

	err := RegisterInitializableDependency(ctx, dep1, "test")
	if err != nil {
		t.Fatalf("Failed to register first dependency: %v", err)
	}

	err = RegisterInitializableDependency(ctx, dep2, "test")
	if err == nil {
		t.Errorf("Expected error when registering duplicate dependency, got nil")
	}
}

func TestGetDependencyByInterface(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testImplementation{}

	err := RegisterInitializableDependency(ctx, dep, "test")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	retrieved, err := GetDependencyByInterface(ctx, TypeOfInterface[testInterface](), "test")
	if err != nil {
		t.Fatalf("Failed to get dependency by interface: %v", err)
	}

	if retrieved != dep {
		t.Errorf("Retrieved dependency is not the same as registered")
	}

	retrievedT, err := GetDependencyByInterfaceType[testInterface](ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency by interface using generic method: %v", err)
	}

	if retrievedT != dep {
		t.Errorf("Retrieved dependency (generic method) is not the same as registered")
	}
}

func TestGetDependencyByQualifier(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	err := RegisterInitializableDependency(ctx, dep, "unique")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	retrieved, err := GetDependencyByQualifier(ctx, "unique")
	if err != nil {
		t.Fatalf("Failed to get dependency by qualifier: %v", err)
	}

	if retrieved != dep {
		t.Errorf("Retrieved dependency is not the same as registered")
	}
}

func TestMustFunctions(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testImplementation{}

	// Test MustRegisterInitializableDependency
	MustRegisterInitializableDependency(ctx, dep, "test")

	// Test MustGetDependency
	retrieved := MustGetDependency(ctx, dep, "test")
	if retrieved != dep {
		t.Errorf("MustGetDependency: Retrieved dependency is not the same as registered")
	}

	// Test MustGetDependencyByInterface
	retrievedByInterface := MustGetDependencyByInterface(ctx, TypeOfInterface[testInterface](), "test")
	if retrievedByInterface != dep {
		t.Errorf("MustGetDependencyByInterface: Retrieved dependency is not the same as registered")
	}

	// Test MustGetDependencyByInterfaceType
	retrievedByInterfaceT := MustGetDependencyByInterfaceType[testInterface](ctx, "test")
	if retrievedByInterfaceT != dep {
		t.Errorf("MustGetDependencyByInterfaceType: Retrieved dependency is not the same as registered")
	}

	// Test MustGetDependencyByQualifier
	retrievedByQualifier := MustGetDependencyByQualifier(ctx, "test")
	if retrievedByQualifier != dep {
		t.Errorf("MustGetDependencyByQualifier: Retrieved dependency is not the same as registered")
	}
}

func TestMustFunctionsPanic(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	// Test MustGetDependency panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetDependency should have panicked")
			}
		}()
		MustGetDependency(ctx, dep, "nonexistent")
	}()

	// Test MustGetDependencyByInterface panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetDependencyByInterface should have panicked")
			}
		}()
		MustGetDependencyByInterface(ctx, TypeOfInterface[testInterface](), "nonexistent")
	}()

	// Test MustGetDependencyByInterfaceType panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetDependencyByInterfaceType should have panicked")
			}
		}()
		MustGetDependencyByInterfaceType[testInterface](ctx, "nonexistent")
	}()

	// Test MustGetDependencyByQualifier panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetDependencyByQualifier should have panicked")
			}
		}()
		MustGetDependencyByQualifier(ctx, "nonexistent")
	}()
}

func TestOptionalQualifier(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep1 := &testDependency{}
	dep2 := &testDependency{}

	// Register dependencies with and without qualifiers
	err := RegisterInitializableDependency(ctx, dep1)
	if err != nil {
		t.Fatalf("Failed to register dependency without qualifier: %v", err)
	}

	err = RegisterInitializableDependency(ctx, dep2, "qualified")
	if err != nil {
		t.Fatalf("Failed to register dependency with qualifier: %v", err)
	}

	// Test getting dependency without qualifier
	retrieved1, err := GetDependency(ctx, &testDependency{})
	if err != nil {
		t.Fatalf("Failed to get dependency without qualifier: %v", err)
	}
	if retrieved1 != dep1 {
		t.Errorf("Retrieved dependency without qualifier is not the same as registered")
	}

	// Test getting dependency with qualifier
	retrieved2, err := GetDependency(ctx, &testDependency{}, "qualified")
	if err != nil {
		t.Fatalf("Failed to get dependency with qualifier: %v", err)
	}
	if retrieved2 != dep2 {
		t.Errorf("Retrieved dependency with qualifier is not the same as registered")
	}
}

func TestGetDependencyT(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	err := RegisterInitializableDependency(ctx, dep, "test")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	retrieved, err := GetDependencyT[*testDependency](ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency using GetDependencyT: %v", err)
	}

	if retrieved != dep {
		t.Errorf("Retrieved dependency is not the same as registered")
	}

	if !retrieved.initialized {
		t.Errorf("Retrieved dependency was not initialized")
	}

	// Test with non-existent dependency
	_, err = GetDependencyT[string](ctx)
	if err == nil {
		t.Errorf("Expected error when getting non-existent dependency, got nil")
	}
}

func TestGetInitializedDependencyT(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	err := RegisterDependencyWithInit(ctx, dep, nil, "test")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	retrieved, err := GetDependencyT[*testDependency](ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency using GetDependencyT: %v", err)
	}

	retrieved, err = GetDependencyT[*testDependency](ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency using GetDependencyT: %v", err)
	}

	if retrieved != dep {
		t.Errorf("Retrieved dependency is not the same as registered")
	}

	if retrieved.initialized {
		t.Errorf("Retrieved dependency was initialized")
	}

	// Test with non-existent dependency
	_, err = GetDependencyT[string](ctx)
	if err == nil {
		t.Errorf("Expected error when getting non-existent dependency, got nil")
	}
}

func TestMustGetDependencyT(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	MustRegisterInitializableDependency(ctx, dep, "test")

	retrieved := MustGetDependencyT[*testDependency](ctx, "test")
	if retrieved != dep {
		t.Errorf("MustGetDependencyT: Retrieved dependency is not the same as registered")
	}

	// Test panic behavior
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetDependencyT should have panicked")
			}
		}()
		MustGetDependencyT[string](ctx)
	}()
}

type cyclicDependency struct {
	name string
	dep  *cyclicDependency
	init func(ctx context.Context)
}

func (c *cyclicDependency) Init(ctx context.Context) {
	if c.init != nil {
		fmt.Printf("Initializing %s\n", c.name) // Debugging line
		c.init(ctx)
		fmt.Printf("Finished initializing %s\n", c.name) // Debugging line
	}
}

func TestDependencyCycleDetection(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	ctx = SetDeadlockTimeout(ctx, 3*time.Second)

	depA := &cyclicDependency{name: "depA"}
	depB := &cyclicDependency{name: "depB"}
	depC := &cyclicDependency{name: "depC"}
	depD := &cyclicDependency{name: "depD"}

	// Create a cycle: depA -> depB -> depC -> depD -> depB
	depA.init = func(ctx context.Context) {
		_, err := GetDependency(ctx, depB, "depB")
		if err != nil {
			panic(err)
		}
	}
	depB.init = func(ctx context.Context) {
		_, err := GetDependency(ctx, depC, "depC")
		if err != nil {
			panic(err)
		}
	}
	depC.init = func(ctx context.Context) {
		_, err := GetDependency(ctx, depD, "depD")
		if err != nil {
			panic(err)
		}
	}
	depD.init = func(ctx context.Context) {
		_, err := GetDependency(ctx, depB, "depB")
		if err != nil {
			panic(err)
		}
	}

	// Register the dependencies
	RegisterInitializableDependency(ctx, depA, "depA")
	RegisterInitializableDependency(ctx, depB, "depB")
	RegisterInitializableDependency(ctx, depC, "depC")
	RegisterInitializableDependency(ctx, depD, "depD")

	// Attempt to initialize depA, which should result in an error due to the circular dependency
	_, err := GetDependency(ctx, depA, "depA")
	if err == nil {
		t.Errorf("Expected error due to dependency cycle, but no error occurred")
	} else if depErr, ok := err.(*DependencyInitializationError); ok {
		if depErr.Message != "Circular dependency detected during initialization" {
			t.Errorf("Expected circular dependency error message, got: %s", depErr.Message)
		}
		expectedStack := []string{
			"*godepinject.cyclicDependency:depA",
			"*godepinject.cyclicDependency:depB",
			"*godepinject.cyclicDependency:depC",
			"*godepinject.cyclicDependency:depD",
			"*godepinject.cyclicDependency:depB",
		}
		if !reflect.DeepEqual(depErr.Stack, expectedStack) {
			t.Errorf("Expected dependency stack %v, but got %v", expectedStack, depErr.Stack)
		}
	} else {
		t.Errorf("Expected DependencyInitializationError, got: %v", err)
	}
}

func TestInitializationTimeout(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	ctx = SetDeadlockTimeout(ctx, 1*time.Second)

	startA := make(chan struct{})
	startB := make(chan struct{})
	depAInitialized := make(chan struct{})
	depBInitialized := make(chan struct{})

	depA := &cyclicDependency{
		name: "depA",
		init: func(ctx context.Context) {
			close(depAInitialized)
			<-startA
			time.Sleep(100 * time.Millisecond)
			_, err := GetDependency(ctx, &cyclicDependency{}, "B")
			if err != nil {
				panic(err)
			}
		},
	}

	depB := &cyclicDependency{
		name: "depB",
		init: func(ctx context.Context) {
			close(depBInitialized)
			<-startB
			time.Sleep(100 * time.Millisecond)
			_, err := GetDependency(ctx, &cyclicDependency{}, "A")
			if err != nil {
				panic(err)
			}
		},
	}

	RegisterInitializableDependency(ctx, depA, "A")
	RegisterInitializableDependency(ctx, depB, "B")

	go func() {
		_, _ = GetDependency(ctx, depA, "A")
	}()

	go func() {
		_, _ = GetDependency(ctx, depB, "B")
	}()

	// Wait for both dependencies to start initializing
	<-depAInitialized
	<-depBInitialized

	// Allow both dependencies to proceed and create a deadlock
	close(startA)
	close(startB)

	// Try to get one of the dependencies, which should now timeout
	_, err := GetDependency(ctx, depA, "A")
	if err == nil {
		t.Errorf("Expected timeout error, but no error occurred")
	} else if depErr, ok := err.(*DependencyInitializationError); ok {
		if !strings.Contains(depErr.Message, "Timeout while waiting to initialize dependency") {
			t.Errorf("Expected timeout error message, got: %s", depErr.Message)
		}
	} else {
		t.Errorf("Expected DependencyInitializationError, got: %v", err)
	}
}

func TestPanicDuringInitialization(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	ctx = SetDeadlockTimeout(ctx, 5*time.Second)

	panicDep := &cyclicDependency{
		name: "panicDep",
		init: func(ctx context.Context) {
			panic("Intentional panic during initialization")
		},
	}

	RegisterInitializableDependency(ctx, panicDep, "panic")

	_, err := GetDependency(ctx, panicDep, "panic")
	if err == nil {
		t.Errorf("Expected error due to panic, but no error occurred")
	} else if depErr, ok := err.(*DependencyInitializationError); ok {
		if !strings.Contains(depErr.Message, "Panic during dependency initialization") {
			t.Errorf("Expected panic error message, got: %s", depErr.Message)
		}
		if !strings.Contains(depErr.Message, "Intentional panic during initialization") {
			t.Errorf("Expected original panic message in error, got: %s", depErr.Message)
		}
	} else {
		t.Errorf("Expected DependencyInitializationError, got: %v", err)
	}
}

func TestMultipleDependenciesImplementingInterface(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	dep1 := &testImplementation{}
	dep2 := &testImplementation{}

	RegisterInitializableDependency(ctx, dep1, "impl1")
	RegisterInitializableDependency(ctx, dep2, "impl2")

	// Try to get dependency without qualifier
	_, err := GetDependencyByInterfaceType[testInterface](ctx)
	if err == nil {
		t.Errorf("Expected error due to multiple implementations, but no error occurred")
	} else if depErr, ok := err.(*DependencyInitializationError); ok {
		if !strings.Contains(depErr.Message, "Multiple dependencies implement the given interface") {
			t.Errorf("Expected multiple implementations error message, got: %s", depErr.Message)
		}
	} else {
		t.Errorf("Expected DependencyInitializationError, got: %v", err)
	}

	// Try to get dependency with qualifier
	retrieved, err := GetDependencyByInterfaceType[testInterface](ctx, "impl1")
	if err != nil {
		t.Errorf("Failed to get dependency with qualifier: %v", err)
	}
	if retrieved != dep1 {
		t.Errorf("Retrieved dependency is not the same as registered")
	}
}

func TestDependencyInitializationOrder(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	initOrder := []string{}
	depA := &cyclicDependency{
		name: "depA",
		init: func(ctx context.Context) {
			initOrder = append(initOrder, "A")
			MustGetDependency(ctx, &cyclicDependency{}, "B")
		},
	}
	depB := &cyclicDependency{
		name: "depB",
		init: func(ctx context.Context) {
			initOrder = append(initOrder, "B")
			MustGetDependency(ctx, &cyclicDependency{}, "C")
		},
	}
	depC := &cyclicDependency{
		name: "depC",
		init: func(ctx context.Context) {
			initOrder = append(initOrder, "C")
		},
	}

	RegisterInitializableDependency(ctx, depA, "A")
	RegisterInitializableDependency(ctx, depB, "B")
	RegisterInitializableDependency(ctx, depC, "C")

	_, err := GetDependency(ctx, depA, "A")
	if err != nil {
		t.Fatalf("Failed to get depA: %v", err)
	}

	expectedOrder := []string{"A", "B", "C"}
	if !reflect.DeepEqual(initOrder, expectedOrder) {
		t.Errorf("Expected initialization order %v, but got %v", expectedOrder, initOrder)
	}
}

func TestConcurrentDependencyInitialization(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	depA := &cyclicDependency{name: "depA"}
	depB := &cyclicDependency{name: "depB"}

	initCounter := 0
	var mu sync.Mutex

	depA.init = func(ctx context.Context) {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		initCounter++
		mu.Unlock()
	}
	depB.init = func(ctx context.Context) {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		initCounter++
		mu.Unlock()
	}

	RegisterInitializableDependency(ctx, depA, "A")
	RegisterInitializableDependency(ctx, depB, "B")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := GetDependency(ctx, depA, "A")
		if err != nil {
			t.Errorf("Failed to get depA: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		_, err := GetDependency(ctx, depB, "B")
		if err != nil {
			t.Errorf("Failed to get depB: %v", err)
		}
	}()

	wg.Wait()

	if initCounter != 2 {
		t.Errorf("Expected 2 initializations, but got %d", initCounter)
	}
}

func TestDependencyNamespaceIsolation(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	ctx2 := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	dep1 := &testDependency{}
	dep2 := &testDependency{}

	RegisterInitializableDependency(ctx1, dep1, "test")
	RegisterInitializableDependency(ctx2, dep2, "test")

	retrieved1, err := GetDependency(ctx1, &testDependency{}, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency from ctx1: %v", err)
	}
	if retrieved1 != dep1 {
		t.Errorf("Retrieved dependency from ctx1 is not the same as registered")
	}

	retrieved2, err := GetDependency(ctx2, &testDependency{}, "test")
	if err != nil {
		t.Fatalf("Failed to get dependency from ctx2: %v", err)
	}
	if retrieved2 != dep2 {
		t.Errorf("Retrieved dependency from ctx2 is not the same as registered")
	}

	if retrieved1 == retrieved2 {
		t.Errorf("Dependencies from different namespaces should not be the same")
	}
}

func TestSetDeadlockTimeout(t *testing.T) {
	baseCtx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	ctx := SetDeadlockTimeout(baseCtx, 500*time.Millisecond)

	startA := make(chan struct{})
	startB := make(chan struct{})
	depAInitialized := make(chan struct{})
	depBInitialized := make(chan struct{})

	depA := &cyclicDependency{
		name: "depA",
		init: func(ctx context.Context) {
			close(depAInitialized)
			<-startA
			time.Sleep(100 * time.Millisecond)
			_, err := GetDependency(ctx, &cyclicDependency{}, "B")
			if err != nil {
				panic(err)
			}
		},
	}

	depB := &cyclicDependency{
		name: "depB",
		init: func(ctx context.Context) {
			close(depBInitialized)
			<-startB
			time.Sleep(100 * time.Millisecond)
			_, err := GetDependency(ctx, &cyclicDependency{}, "A")
			if err != nil {
				panic(err)
			}
		},
	}

	RegisterInitializableDependency(ctx, depA, "A")
	RegisterInitializableDependency(ctx, depB, "B")

	go func() {
		_, _ = GetDependency(ctx, depA, "A")
	}()

	go func() {
		_, _ = GetDependency(ctx, depB, "B")
	}()

	// Wait for both dependencies to start initializing
	<-depAInitialized
	<-depBInitialized

	// Allow both dependencies to proceed and create a deadlock
	close(startA)
	close(startB)

	// Try to get one of the dependencies, which should now timeout
	_, err := GetDependency(ctx, depA, "A")
	if err == nil {
		t.Errorf("Expected timeout error, but no error occurred")
	} else if depErr, ok := err.(*DependencyInitializationError); ok {
		if !strings.Contains(depErr.Message, "Timeout while waiting to initialize dependency") {
			t.Errorf("Expected timeout error message, got: %s", depErr.Message)
		}
	} else {
		t.Errorf("Expected DependencyInitializationError, got: %v", err)
	}
}

func TestNestedDependencyRegistration(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	depA := &cyclicDependency{
		name: "depA",
		init: func(ctx context.Context) {
			// Register a new dependency inside the initialization of depA
			depC := &cyclicDependency{name: "depC"}
			err := RegisterInitializableDependency(ctx, depC, "C")
			if err != nil {
				t.Errorf("Failed to register depC inside depA initialization: %v", err)
			}
		},
	}

	depB := &cyclicDependency{
		name: "depB",
		init: func(ctx context.Context) {
			// Try to get depC, which should be registered by depA
			_, err := GetDependency(ctx, &cyclicDependency{}, "C")
			if err != nil {
				t.Errorf("Failed to get depC from depB initialization: %v", err)
			}
		},
	}

	// Register depA and depB
	err := RegisterInitializableDependency(ctx, depA, "A")
	if err != nil {
		t.Fatalf("Failed to register depA: %v", err)
	}

	err = RegisterInitializableDependency(ctx, depB, "B")
	if err != nil {
		t.Fatalf("Failed to register depB: %v", err)
	}

	// This should trigger the initialization of depA, which registers depC,
	// and then initialize depB, which tries to get depC
	_, err = GetDependency(ctx, depA, "A")
	if err != nil {
		t.Fatalf("Failed to get depA: %v", err)
	}

	_, err = GetDependency(ctx, depB, "B")
	if err != nil {
		t.Fatalf("Failed to get depB: %v", err)
	}

	// Verify that depC is registered and can be retrieved
	_, err = GetDependency(ctx, &cyclicDependency{}, "C")
	if err != nil {
		t.Fatalf("Failed to get depC after nested registration: %v", err)
	}
}

func TestDeleteAllDependencies(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	err := RegisterInitializableDependency(ctx, dep, "test")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	err = DeleteAllDependencies(ctx)
	if err != nil {
		t.Fatalf("Failed to delete all dependencies: %v", err)
	}

	_, err = GetDependency(ctx, dep, "test")
	if err == nil || !errors.Is(err, ErrDependencyNotFound) {
		t.Errorf("Expected error when getting dependency after deletion, got %v", err)
	}
}

func TestRegisterDependency(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	// Create a simple struct to use as a dependency
	type simpleDep struct {
		Value string
	}

	dep := &simpleDep{Value: "test value"}

	// Register the dependency without an initializer
	err := RegisterDependency(ctx, dep, "simple")
	if err != nil {
		t.Fatalf("Failed to register dependency: %v", err)
	}

	// Retrieve the dependency
	retrieved, err := GetDependency(ctx, &simpleDep{}, "simple")
	if err != nil {
		t.Fatalf("Failed to get dependency: %v", err)
	}

	// Check if the retrieved dependency is the same as the registered one
	if retrieved != dep {
		t.Errorf("Retrieved dependency is not the same as registered")
	}

	// Check if the Value field is correct
	if retrievedDep, ok := retrieved.(*simpleDep); ok {
		if retrievedDep.Value != "test value" {
			t.Errorf("Expected Value to be 'test value', got '%s'", retrievedDep.Value)
		}
	} else {
		t.Errorf("Retrieved dependency is not of type *simpleDep")
	}

	// Try to register the same dependency again (should fail)
	err = RegisterDependency(ctx, &simpleDep{}, "simple")
	if err == nil {
		t.Errorf("Expected error when registering duplicate dependency, got nil")
	}

	// Register another dependency with a different qualifier
	dep2 := &simpleDep{Value: "another value"}
	err = RegisterDependency(ctx, dep2, "simple2")
	if err != nil {
		t.Fatalf("Failed to register second dependency: %v", err)
	}

	// Retrieve the second dependency
	retrieved2, err := GetDependency(ctx, &simpleDep{}, "simple2")
	if err != nil {
		t.Fatalf("Failed to get second dependency: %v", err)
	}

	// Check if the second retrieved dependency is correct
	if retrieved2 != dep2 {
		t.Errorf("Retrieved second dependency is not the same as registered")
	}
}

func TestListRegisteredDependencies(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

	// Register some dependencies
	RegisterDependency(ctx, &testDependency{}, "test1")
	RegisterDependency(ctx, &testImplementation{}, "test2")
	RegisterDependency(ctx, "string dep", "test3")
	RegisterDependency(ctx, 42, "test4")
	RegisterDependency(ctx, &testDependency{}, "test5") // Same type as test1, different qualifier

	// List registered dependencies
	dependencyIDs, err := ListRegisteredDependencies(ctx)
	if err != nil {
		t.Fatalf("Failed to list registered dependencies: %v", err)
	}

	// Expected dependencyIDs
	expectedDependencyIDs := []string{
		"*godepinject.testDependency:test1",
		"*godepinject.testDependency:test5",
		"*godepinject.testImplementation:test2",
		"int:test4",
		"string:test3",
	}

	// Check if the returned dependencyIDs match the expected dependencyIDs
	if !reflect.DeepEqual(dependencyIDs, expectedDependencyIDs) {
		t.Errorf("Expected dependencyIDs %v, but got %v", expectedDependencyIDs, dependencyIDs)
	}

	// Test with non-existent namespace
	invalidCtx := context.WithValue(context.Background(), DependencyNamespaceKey, "non-existent")
	_, err = ListRegisteredDependencies(invalidCtx)
	if err == nil {
		t.Error("Expected error for non-existent namespace, but got nil")
	}

	// Test with missing namespace key
	invalidCtx = context.Background()
	_, err = ListRegisteredDependencies(invalidCtx)
	if err == nil {
		t.Error("Expected error for missing namespace key, but got nil")
	}
}

// TestInterface is an interface used for testing dependency injection
type TestInterface interface {
	DoSomething() string
}

// TestImpl is a concrete implementation of TestInterface
type TestImpl struct {
	value string
}

// DoSomething implements the TestInterface
func (t *TestImpl) DoSomething() string {
	return t.value
}

func TestGetDependencyByInterfaceCache(t *testing.T) {
	// Test 1: Basic caching
	t.Run("Basic caching", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
		dep := &TestImpl{value: "dep"}
		err := RegisterDependency(ctx, dep)
		if err != nil {
			t.Fatalf("Failed to register dependency: %v", err)
		}

		// First call should not use cache
		start := time.Now()
		retrieved1, err := GetDependencyByInterfaceType[TestInterface](ctx)
		if err != nil {
			t.Fatalf("Failed to get dependency: %v", err)
		}
		firstCallDuration := time.Since(start)

		// Second call should use cache and be faster
		start = time.Now()
		retrieved2, err := GetDependencyByInterfaceType[TestInterface](ctx)
		if err != nil {
			t.Fatalf("Failed to get dependency: %v", err)
		}
		secondCallDuration := time.Since(start)

		if retrieved1 != retrieved2 {
			t.Errorf("Retrieved dependencies are not the same instance")
		}
		if secondCallDuration >= firstCallDuration {
			t.Errorf("Second call was not faster than first call, caching might not be working")
		}
	})

	// Test 2: Caching behavior with multiple dependencies
	t.Run("Caching behavior with multiple dependencies", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

		dep1 := &TestImpl{value: "dep1"}
		err := RegisterDependency(ctx, dep1, "first")
		if err != nil {
			t.Fatalf("Failed to register first dependency: %v", err)
		}

		dep2 := &TestImpl{value: "dep2"}
		err = RegisterDependency(ctx, dep2, "second")
		if err != nil {
			t.Fatalf("Failed to register second dependency: %v", err)
		}

		// First call for "first" dependency
		start := time.Now()
		retrieved1, err := GetDependencyByInterfaceType[TestInterface](ctx, "first")
		if err != nil {
			t.Fatalf("Failed to get first dependency: %v", err)
		}
		firstCallDuration := time.Since(start)

		// Second call for "first" dependency (should be cached)
		start = time.Now()
		retrievedAgain1, err := GetDependencyByInterfaceType[TestInterface](ctx, "first")
		if err != nil {
			t.Fatalf("Failed to get first dependency again: %v", err)
		}
		secondCallDuration := time.Since(start)

		// Verify that the second call was faster (indicating cache usage)
		if secondCallDuration >= firstCallDuration {
			t.Errorf("Second call for 'first' dependency was not faster. First: %v, Second: %v", firstCallDuration, secondCallDuration)
		}

		// Verify that both calls returned the same instance
		if retrieved1 != retrievedAgain1 {
			t.Errorf("Retrieved 'first' dependency instances are not the same")
		}

		// First call for "second" dependency
		start = time.Now()
		retrieved2, err := GetDependencyByInterfaceType[TestInterface](ctx, "second")
		if err != nil {
			t.Fatalf("Failed to get second dependency: %v", err)
		}
		firstCallDuration2 := time.Since(start)

		// Second call for "second" dependency (should be cached)
		start = time.Now()
		retrievedAgain2, err := GetDependencyByInterfaceType[TestInterface](ctx, "second")
		if err != nil {
			t.Fatalf("Failed to get second dependency again: %v", err)
		}
		secondCallDuration2 := time.Since(start)

		// Verify that the second call was faster (indicating cache usage)
		if secondCallDuration2 >= firstCallDuration2 {
			t.Errorf("Second call for 'second' dependency was not faster. First: %v, Second: %v", firstCallDuration2, secondCallDuration2)
		}

		// Verify that both calls returned the same instance
		if retrieved2 != retrievedAgain2 {
			t.Errorf("Retrieved 'second' dependency instances are not the same")
		}

		// Verify that the two dependencies are different
		if retrieved1 == retrieved2 {
			t.Errorf("Retrieved dependencies are the same instance")
		}

		// Verify the values of the dependencies
		if retrieved1.DoSomething() != "dep1" {
			t.Errorf("First dependency has incorrect value. Expected 'dep1', got '%s'", retrieved1.DoSomething())
		}
		if retrieved2.DoSomething() != "dep2" {
			t.Errorf("Second dependency has incorrect value. Expected 'dep2', got '%s'", retrieved2.DoSomething())
		}
	})

	// Test 3: Separate caches for different namespaces
	t.Run("Separate caches for different namespaces", func(t *testing.T) {
		ctx1 := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
		ctx2 := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
		dep1 := &TestImpl{value: "dep1"}
		dep2 := &TestImpl{value: "dep2"}
		err := RegisterDependency(ctx1, dep1)
		if err != nil {
			t.Fatalf("Failed to register dependency in ctx1: %v", err)
		}
		err = RegisterDependency(ctx2, dep2)
		if err != nil {
			t.Fatalf("Failed to register dependency in ctx2: %v", err)
		}

		retrieved1, err := GetDependencyByInterfaceType[TestInterface](ctx1)
		if err != nil {
			t.Fatalf("Failed to get dependency from ctx1: %v", err)
		}
		retrieved2, err := GetDependencyByInterfaceType[TestInterface](ctx2)
		if err != nil {
			t.Fatalf("Failed to get dependency from ctx2: %v", err)
		}

		if retrieved1 == retrieved2 {
			t.Errorf("Retrieved dependencies from different namespaces are the same instance")
		}
		if retrieved1.DoSomething() != "dep1" || retrieved2.DoSomething() != "dep2" {
			t.Errorf("Retrieved dependencies have incorrect values")
		}
	})

	// Test 4: Cache invalidation on DeleteAllDependencies
	t.Run("Cache invalidation on DeleteAllDependencies", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
		dep := &TestImpl{value: "dep"}
		err := RegisterDependency(ctx, dep)
		if err != nil {
			t.Fatalf("Failed to register dependency: %v", err)
		}

		// First call to cache the dependency
		_, err = GetDependencyByInterfaceType[TestInterface](ctx)
		if err != nil {
			t.Fatalf("Failed to get dependency: %v", err)
		}

		// Delete all dependencies, which should invalidate the cache
		err = DeleteAllDependencies(ctx)
		if err != nil {
			t.Fatalf("Failed to delete all dependencies: %v", err)
		}

		// This call should not find the dependency
		_, err = GetDependencyByInterfaceType[TestInterface](ctx)
		if err == nil {
			t.Errorf("Expected error after deleting all dependencies, but got nil")
		}
	})

	// Test 5: Caching with qualifiers
	t.Run("Caching with qualifiers", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

		dep1 := &TestImpl{value: "dep1"}
		dep2 := &TestImpl{value: "dep2"}
		err := RegisterDependency(ctx, dep1, "qualifier1")
		if err != nil {
			t.Fatalf("Failed to register dependency with qualifier1: %v", err)
		}
		err = RegisterDependency(ctx, dep2, "qualifier2")
		if err != nil {
			t.Fatalf("Failed to register dependency with qualifier2: %v", err)
		}

		// First calls to retrieve the dependencies
		start := time.Now()
		retrieved1, err := GetDependencyByInterfaceType[TestInterface](ctx, "qualifier1")
		if err != nil {
			t.Fatalf("Failed to get dependency with qualifier1: %v", err)
		}
		firstCallDuration1 := time.Since(start)

		start = time.Now()
		retrieved2, err := GetDependencyByInterfaceType[TestInterface](ctx, "qualifier2")
		if err != nil {
			t.Fatalf("Failed to get dependency with qualifier2: %v", err)
		}
		firstCallDuration2 := time.Since(start)

		if retrieved1 == retrieved2 {
			t.Errorf("Retrieved dependencies with different qualifiers are the same instance")
		}

		// Second calls should use cache and be faster
		start = time.Now()
		cachedRetrieved1, err := GetDependencyByInterfaceType[TestInterface](ctx, "qualifier1")
		if err != nil {
			t.Fatalf("Failed to get cached dependency with qualifier1: %v", err)
		}
		secondCallDuration1 := time.Since(start)

		start = time.Now()
		cachedRetrieved2, err := GetDependencyByInterfaceType[TestInterface](ctx, "qualifier2")
		if err != nil {
			t.Fatalf("Failed to get cached dependency with qualifier2: %v", err)
		}
		secondCallDuration2 := time.Since(start)

		// Verify that second calls were faster (indicating cache usage)
		if secondCallDuration1 >= firstCallDuration1 {
			t.Errorf("Second call for qualifier1 was not faster. First: %v, Second: %v", firstCallDuration1, secondCallDuration1)
		}
		if secondCallDuration2 >= firstCallDuration2 {
			t.Errorf("Second call for qualifier2 was not faster. First: %v, Second: %v", firstCallDuration2, secondCallDuration2)
		}

		// Verify that cached retrievals return the same instances
		if retrieved1 != cachedRetrieved1 {
			t.Errorf("Cached retrieval with qualifier1 returned a different instance")
		}
		if retrieved2 != cachedRetrieved2 {
			t.Errorf("Cached retrieval with qualifier2 returned a different instance")
		}

		// Verify the values of the dependencies
		if retrieved1.DoSomething() != "dep1" {
			t.Errorf("Dependency with qualifier1 has incorrect value. Expected 'dep1', got '%s'", retrieved1.DoSomething())
		}
		if retrieved2.DoSomething() != "dep2" {
			t.Errorf("Dependency with qualifier2 has incorrect value. Expected 'dep2', got '%s'", retrieved2.DoSomething())
		}

		// Test retrieval without a qualifier (should fail or return a specific dependency if that's the expected behavior)
		_, err = GetDependencyByInterfaceType[TestInterface](ctx)
		if err == nil {
			t.Errorf("Expected an error when retrieving without a qualifier, but got nil")
		}
	})

	// Test 6: Caching behavior with and without qualifiers
	t.Run("Caching with and without qualifiers", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())

		// Register a dependency without a qualifier
		depWithoutQualifier := &TestImpl{value: "depWithoutQualifier"}
		err := RegisterDependency(ctx, depWithoutQualifier)
		if err != nil {
			t.Fatalf("Failed to register dependency without qualifier: %v", err)
		}

		// Retrieve without qualifier
		retrieved1, err := GetDependencyByInterfaceType[TestInterface](ctx)
		if err != nil {
			t.Fatalf("Failed to get dependency without qualifier: %v", err)
		}

		// Retrieve with empty qualifier (should be the same as without qualifier)
		retrieved2, err := GetDependencyByInterfaceType[TestInterface](ctx, "")
		if err != nil {
			t.Fatalf("Failed to get dependency with empty qualifier: %v", err)
		}

		if retrieved1 != retrieved2 {
			t.Errorf("Retrieved dependencies with and without qualifier are not the same instance")
		}

		// Register a new dependency with a qualifier
		depWithQualifier := &TestImpl{value: "depWithQualifier"}
		err = RegisterDependency(ctx, depWithQualifier, "qualifier")
		if err != nil {
			t.Fatalf("Failed to register dependency with qualifier: %v", err)
		}

		// Attempt to retrieve without qualifier (should now fail due to multiple implementations)
		_, err = GetDependencyByInterfaceType[TestInterface](ctx)
		if err == nil {
			t.Errorf("Expected error when getting dependency without qualifier after registering multiple dependencies, but got nil")
		} else if !strings.Contains(err.Error(), "Multiple dependencies implement the given interface") {
			t.Errorf("Unexpected error when getting dependency without qualifier: %v", err)
		}

		// Retrieve the qualified dependency
		retrievedQualified, err := GetDependencyByInterfaceType[TestInterface](ctx, "qualifier")
		if err != nil {
			t.Fatalf("Failed to get dependency with qualifier: %v", err)
		}
		if retrievedQualified == retrieved1 {
			t.Errorf("Retrieved qualified dependency is the same as unqualified dependency")
		}

		// Verify the values of the dependencies
		if retrieved1.DoSomething() != "depWithoutQualifier" {
			t.Errorf("Unqualified dependency has incorrect value. Expected 'depWithoutQualifier', got '%s'", retrieved1.DoSomething())
		}
		if retrievedQualified.DoSomething() != "depWithQualifier" {
			t.Errorf("Qualified dependency has incorrect value. Expected 'depWithQualifier', got '%s'", retrievedQualified.DoSomething())
		}

		// Test caching for qualified dependency
		start := time.Now()
		_, err = GetDependencyByInterfaceType[TestInterface](ctx, "qualifier")
		if err != nil {
			t.Fatalf("Failed to get qualified dependency for caching test: %v", err)
		}
		firstCallDuration := time.Since(start)

		start = time.Now()
		_, err = GetDependencyByInterfaceType[TestInterface](ctx, "qualifier")
		if err != nil {
			t.Fatalf("Failed to get qualified dependency for caching test (second call): %v", err)
		}
		secondCallDuration := time.Since(start)

		if secondCallDuration >= firstCallDuration {
			t.Errorf("Second call for qualified dependency was not faster. First: %v, Second: %v", firstCallDuration, secondCallDuration)
		}
	})
}
