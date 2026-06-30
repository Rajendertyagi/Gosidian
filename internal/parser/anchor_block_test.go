package parser

import (
	"reflect"
	"testing"
)

func TestExtractFrontmatterBlock_InlineTools(t *testing.T) {
	raw := `title: Foo
type: agent
harness:
  name: frontend-engineer
  description: "Vue 3 lib, gate generalizzabile?"
  tools: [Read, Edit, mcp__gosidian__memory_get]
  model: sonnet
tags: [x, type:agent]`
	got := ExtractFrontmatterBlock(raw, "harness")
	if got == nil {
		t.Fatal("expected a block, got nil")
	}
	if got["name"] != "frontend-engineer" {
		t.Errorf("name = %v", got["name"])
	}
	if got["description"] != "Vue 3 lib, gate generalizzabile?" {
		t.Errorf("description = %v", got["description"])
	}
	if got["model"] != "sonnet" {
		t.Errorf("model = %v", got["model"])
	}
	want := []string{"Read", "Edit", "mcp__gosidian__memory_get"}
	if !reflect.DeepEqual(got["tools"], want) {
		t.Errorf("tools = %v, want %v", got["tools"], want)
	}
}

func TestExtractFrontmatterBlock_BlockListTools(t *testing.T) {
	raw := `harness:
  name: x
  tools:
    - Read
    - Bash
type: agent`
	got := ExtractFrontmatterBlock(raw, "harness")
	if got == nil {
		t.Fatal("nil")
	}
	want := []string{"Read", "Bash"}
	if !reflect.DeepEqual(got["tools"], want) {
		t.Errorf("tools = %v, want %v", got["tools"], want)
	}
}

func TestExtractFrontmatterBlock_Absent(t *testing.T) {
	if got := ExtractFrontmatterBlock("title: Foo\ntype: agent", "harness"); got != nil {
		t.Errorf("expected nil for absent block, got %v", got)
	}
}

func TestExtractFrontmatterBlock_InlineValueIsNotBlock(t *testing.T) {
	if got := ExtractFrontmatterBlock("harness: something\ntype: agent", "harness"); got != nil {
		t.Errorf("expected nil for inline value, got %v", got)
	}
}
