package retrieve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The download path and the presence check must resolve the SAME model. They
// used to name miniLMInt8 in two places; a swap updating one and not the other
// would download one model and then look for another, and the hook's only
// symptom would be a semantic arm that silently never runs.
//
// Exercised by actually placing what ensureModel would have downloaded — at the
// pinned sizes, as sparse files so a 118 MB pin costs no disk — and asking the
// presence check to find it. Comparing the two directory expressions instead
// would just restate the same line twice and pass no matter what.
func TestPresenceCheckFindsWhatEnsureWouldDownload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, ok := ModelDirIfPresent(); ok {
		t.Fatal("an empty models root must not report the model present")
	}

	dir, err := ensureModel(t.Context(), modelSpec{dir: activeModel.dir}) // creates the dir, downloads nothing
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range activeModel.files {
		fh, err := os.Create(filepath.Join(dir, f.name))
		if err != nil {
			t.Fatal(err)
		}
		if err := fh.Truncate(f.size); err != nil { // sparse: the size is what is checked
			t.Fatal(err)
		}
		fh.Close()
	}

	got, ok := ModelDirIfPresent()
	if !ok {
		t.Fatal("presence check did not find the files ensureModel's directory holds")
	}
	if got != dir {
		t.Fatalf("presence dir %q != ensure dir %q", got, dir)
	}

	// And a truncated file must fail the check rather than be used unverified.
	if err := os.Truncate(filepath.Join(dir, activeModel.files[0].name), activeModel.files[0].size-1); err != nil {
		t.Fatal(err)
	}
	if _, ok := ModelDirIfPresent(); ok {
		t.Fatal("a short file must not count as present")
	}
}

// Every pinned file must be fully specified and fetched over https — a spec with
// a blank hash or an http URL would download unverified bytes.
func TestActiveModelIsFullyPinned(t *testing.T) {
	for _, f := range activeModel.files {
		if f.name == "" || f.size <= 0 || len(f.sha256) != 64 {
			t.Fatalf("model file %+v is not fully pinned (name/size/sha256)", f)
		}
		if !strings.HasPrefix(f.url, "https://") {
			t.Fatalf("model file %q must be fetched over https, got %q", f.name, f.url)
		}
	}
}

// hugot loads a model directory by fixed filenames; a spec that renamed one
// would download successfully and then fail at pipeline build, where retrieval
// fails open — i.e. silently.
func TestActiveModelShipsTheFilenamesHugotExpects(t *testing.T) {
	need := map[string]bool{"model.onnx": false, "tokenizer.json": false, "config.json": false}
	for _, f := range activeModel.files {
		if _, ok := need[f.name]; ok {
			need[f.name] = true
		}
	}
	for name, present := range need {
		if !present {
			t.Errorf("active model does not ship %s", name)
		}
	}
}

// The whole point of the swap: the active model must be the multilingual one.
// Stated as an assertion so reverting the pointer without reverting the index
// format version cannot pass quietly.
func TestActiveModelIsMultilingual(t *testing.T) {
	if activeModel.dir != multilingualInt8.dir {
		t.Fatalf("active model is %q, want the multilingual spec %q", activeModel.dir, multilingualInt8.dir)
	}
	// Stated explicitly rather than implied: reverting to the English-only spec
	// without also reverting indexFormatVersion would leave every user's index
	// unreadable by a binary that then rebuilds it with the wrong model.
	if activeModel.dir == miniLMInt8.dir {
		t.Fatal("active model fell back to the English-only spec")
	}
}
