package generation

import "testing"

func TestBumpAndCurrent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if n, _ := Current(); n != 0 {
		t.Errorf("initial = %d, want 0", n)
	}
	if err := Bump(); err != nil {
		t.Fatal(err)
	}
	if err := Bump(); err != nil {
		t.Fatal(err)
	}
	if n, _ := Current(); n != 2 {
		t.Errorf("after 2 bumps = %d, want 2", n)
	}
}

func TestChangedAndMarkSeen(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()

	// global never bumped, project unseen: 0 == 0 → not changed (no global → no fan-out)
	if ch, _, _ := Changed(proj); ch {
		t.Error("0 vs 0 should not be changed")
	}
	if err := Bump(); err != nil { // global advances to 1
		t.Fatal(err)
	}
	ch, cur, _ := Changed(proj)
	if !ch || cur != 1 {
		t.Errorf("after bump: changed=%v cur=%d, want true/1", ch, cur)
	}
	if err := MarkSeen(proj, cur); err != nil {
		t.Fatal(err)
	}
	if ch, _, _ := Changed(proj); ch {
		t.Error("after MarkSeen should not be changed")
	}
	if err := Bump(); err != nil {
		t.Fatal(err)
	}
	if ch, _, _ := Changed(proj); !ch {
		t.Error("after another bump should be changed again")
	}
}
