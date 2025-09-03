package compose

import (
	"context"
	"testing"
)

// --- Package-level types to trigger assignableTypeMay ---
type Fooer interface{ Foo() string }
type FooImpl struct{}

func (FooImpl) Foo() string { return "foo" }

// For field mapping: predecessor field is an interface, successor field is a concrete type
// that implements the interface, making checkAssignable return `May`.
type Animal interface{ Speak() string }
type Dog struct{}

func (Dog) Speak() string { return "woof" }

type A struct { // predecessor output
	Pet Animal
}
type B struct { // successor input
	Pet Dog
}

// Demonstrates AddNode/AddEdge/Branch to compile, including a field mapping that triggers
// a May -> runtime converter injection.
func TestGraph_Minimal_AddNode_AddEdge_Branch_WithFieldMapping_May(t *testing.T) {
	ctx := context.Background()

	// --- Build graph ---
	g := NewGraph[string, map[string]any]()

	// Node: START -> makeFoo (string -> Fooer)
	makeFoo := InvokableLambda(func(ctx context.Context, in string) (Fooer, error) {
		return FooImpl{}, nil
	})
	if err := g.AddLambdaNode("makeFoo", makeFoo); err != nil {
		t.Fatalf("AddLambdaNode makeFoo: %v", err)
	}

	// Node: useFoo (FooImpl -> map[string]any), edge from makeFoo.
	// This edge triggers assignableTypeMay (predecessor output is interface Fooer, successor input is concrete FooImpl).
	useFoo := InvokableLambda(func(ctx context.Context, in FooImpl) (map[string]any, error) {
		return map[string]any{"use": in.Foo()}, nil
	})
	if err := g.AddLambdaNode("useFoo", useFoo); err != nil {
		t.Fatalf("AddLambdaNode useFoo: %v", err)
	}

	// Node: makeA (Fooer -> A)
	makeA := InvokableLambda(func(ctx context.Context, in Fooer) (A, error) {
		return A{Pet: Dog{}}, nil // dynamic type Dog implements Animal
	})
	if err := g.AddLambdaNode("makeA", makeA); err != nil {
		t.Fatalf("AddLambdaNode makeA: %v", err)
	}

	// Node: needB (B -> map[string]any)
	needB := InvokableLambda(func(ctx context.Context, in B) (map[string]any, error) {
		return map[string]any{"need": in.Pet.Speak()}, nil
	})
	if err := g.AddLambdaNode("needB", needB); err != nil {
		t.Fatalf("AddLambdaNode needB: %v", err)
	}

	// Node: skip (Fooer -> map[string]any)
	skip := InvokableLambda(func(ctx context.Context, in Fooer) (map[string]any, error) {
		return map[string]any{"skip": "ok"}, nil
	})
	if err := g.AddLambdaNode("skip", skip); err != nil {
		t.Fatalf("AddLambdaNode skip: %v", err)
	}

	// Edges
	if err := g.AddEdge(START, "makeFoo"); err != nil {
		t.Fatalf("AddEdge START->makeFoo: %v", err)
	}
	if err := g.AddEdge("makeFoo", "useFoo"); err != nil {
		t.Fatalf("AddEdge makeFoo->useFoo: %v", err)
	}

	// Branch on makeFoo: choose "makeA" (always), or could go "skip".
	branch := NewGraphBranch(func(ctx context.Context, in Fooer) (string, error) {
		return "makeA", nil
	}, map[string]bool{"makeA": true, "skip": true})
	if err := g.AddBranch("makeFoo", branch); err != nil {
		t.Fatalf("AddBranch makeFoo: %v", err)
	}

	// Field mapping: A.Pet(interface) -> B.Pet(concrete Dog), this yields assignableTypeMay
	// and injects a per-field runtime checker, plus mapping converter.
	if err := g.graph.addEdgeWithMappings("makeA", "needB", false, false, MapFields("Pet", "Pet")); err != nil {
		t.Fatalf("addEdgeWithMappings makeA->needB: %v", err)
	}

	// End edges
	if err := g.AddEdge("useFoo", END); err != nil {
		t.Fatalf("AddEdge useFoo->END: %v", err)
	}
	if err := g.AddEdge("needB", END); err != nil {
		t.Fatalf("AddEdge needB->END: %v", err)
	}
	if err := g.AddEdge("skip", END); err != nil {
		t.Fatalf("AddEdge skip->END: %v", err)
	}

	// Hook compile callback to observe GraphInfo (mappings etc.)
	gotInfo := make(chan *GraphInfo, 1)
	r, err := g.Compile(ctx, WithGraphName("min-example"), WithNodeTriggerMode(AllPredecessor), WithGraphCompileCallbacks(GraphCompileCallbackFunc(func(ctx context.Context, info *GraphInfo) {
		gotInfo <- info
	})))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// Run
	out, err := r.Invoke(ctx, "input")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	// Expect merged map from two paths: useFoo (via May converter on edge) and needB (via field mapping + checker)
	if out["use"] != "foo" {
		t.Fatalf("unexpected 'use' value: %v", out["use"])
	}
	if out["need"] != "woof" {
		t.Fatalf("unexpected 'need' value: %v", out["need"])
	}

	// Optionally assert compile info arrived and contains recorded mappings for node needB.
	select {
	case info := <-gotInfo:
		node, ok := info.Nodes["needB"]
		if !ok {
			t.Fatalf("GraphInfo missing node 'needB'")
		}
		if len(node.Mappings) == 0 {
			t.Fatalf("expected field mappings recorded for needB, got none")
		}
	default:
		// not fatal; callback registration is best-effort in this test context
	}
}

// GraphCompileCallbackFunc is a helper to use a func as GraphCompileCallback.
type GraphCompileCallbackFunc func(ctx context.Context, info *GraphInfo)

func (f GraphCompileCallbackFunc) OnFinish(ctx context.Context, info *GraphInfo) { f(ctx, info) }
