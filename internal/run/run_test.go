package run

import "testing"

func TestTypeMarksCharStates(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"go"})
	r.Type('g')
	r.Type('x') // wrong
	states := r.States()
	if states[0] != Correct {
		t.Errorf("char 0: got %v, want Correct", states[0])
	}
	if states[1] != Incorrect {
		t.Errorf("char 1: got %v, want Incorrect", states[1])
	}
	if r.Cursor() != 2 {
		t.Errorf("cursor: got %d, want 2", r.Cursor())
	}
}

func TestBackspaceResetsChar(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"go"})
	r.Type('g')
	r.Backspace()
	if r.Cursor() != 0 {
		t.Errorf("cursor: got %d, want 0", r.Cursor())
	}
	if r.States()[0] != Untyped {
		t.Errorf("char 0: got %v, want Untyped", r.States()[0])
	}
}

func TestBackspaceAtStartIsNoop(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"go"})
	r.Backspace()
	if r.Cursor() != 0 {
		t.Errorf("cursor: got %d, want 0", r.Cursor())
	}
}

func TestTypePastEndIsNoop(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"a"})
	r.Type('a')
	r.Type('b') // past end
	if r.Cursor() != 1 {
		t.Errorf("cursor: got %d, want 1", r.Cursor())
	}
}

func TestAppendWordsJoinsWithSpaces(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"one", "two"})
	if string(r.Text()) != "one two" {
		t.Errorf("text: got %q, want %q", string(r.Text()), "one two")
	}
}
