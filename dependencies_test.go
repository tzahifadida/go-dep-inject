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

	err := RegisterDependency(ctx, dep, "test")
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

	err := RegisterDependency(ctx, dep1, "test")
	if err != nil {
		t.Fatalf("Failed to register first dependency: %v", err)
	}

	err = RegisterDependency(ctx, dep2, "test")
	if err == nil {
		t.Errorf("Expected error when registering duplicate dependency, got nil")
	}
}

func TestGetDependencyByInterface(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testImplementation{}

	err := RegisterDependency(ctx, dep, "test")
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

	err := RegisterDependency(ctx, dep, "unique")
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

	// Test MustRegisterDependency
	MustRegisterDependency(ctx, dep, "test")

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
	err := RegisterDependency(ctx, dep1)
	if err != nil {
		t.Fatalf("Failed to register dependency without qualifier: %v", err)
	}

	err = RegisterDependency(ctx, dep2, "qualified")
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

	err := RegisterDependency(ctx, dep, "test")
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

func TestMustGetDependencyT(t *testing.T) {
	ctx := context.WithValue(context.Background(), DependencyNamespaceKey, uuid.New().String())
	dep := &testDependency{}

	MustRegisterDependency(ctx, dep, "test")

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
	RegisterDependency(ctx, depA, "depA")
	RegisterDependency(ctx, depB, "depB")
	RegisterDependency(ctx, depC, "depC")
	RegisterDependency(ctx, depD, "depD")

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

	RegisterDependency(ctx, depA, "A")
	RegisterDependency(ctx, depB, "B")

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

	RegisterDependency(ctx, panicDep, "panic")

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

	RegisterDependency(ctx, dep1, "impl1")
	RegisterDependency(ctx, dep2, "impl2")

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

	RegisterDependency(ctx, depA, "A")
	RegisterDependency(ctx, depB, "B")
	RegisterDependency(ctx, depC, "C")

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

	RegisterDependency(ctx, depA, "A")
	RegisterDependency(ctx, depB, "B")

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

	RegisterDependency(ctx1, dep1, "test")
	RegisterDependency(ctx2, dep2, "test")

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

	RegisterDependency(ctx, depA, "A")
	RegisterDependency(ctx, depB, "B")

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
			err := RegisterDependency(ctx, depC, "C")
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
	err := RegisterDependency(ctx, depA, "A")
	if err != nil {
		t.Fatalf("Failed to register depA: %v", err)
	}

	err = RegisterDependency(ctx, depB, "B")
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
