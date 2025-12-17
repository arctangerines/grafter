package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func dumb_assert[V comparable](a V, b V, str string) error {
	fmt.Printf("%s", str)
	if a != b {
		return fmt.Errorf("ASSERTION FAILED: %v != %v", a, b)
	} else {
		fmt.Printf("ASSERT: %v == %v\n", a, b)
		return nil
	}
}

const test_string = "The world was so recent that many things lacked names, and in order to indicate them it was necessary to point."
const test_string2 = `The children would remember for the rest of their lives the august solemnity with which their father, devastated by his prolonged vigil and by the wrath of his imagination, revealed his discovery to them:

‘The earth is round, like an orange.’`

// XXX: Interesting thing happened, when creating the file, writing to it, using the same
// file* and doing io.copy to the hash
// then doing it again with all the b variables, kinda generates garbage?
// it has to do with buffering im 90% sure
// need to explore that pls
// open file to write
// write to it
// hash it
// close it
// TODO: A function that tests for files that do not exist
// FIXME: Need to test with home directory as well
// FIXME: Need to test duplicate
func TestFilesExist(t *testing.T) {
	var err error
	tmp_dir := "tmp_grafter_tests"
	tmp_dir_src := "tmp_grafter_tests/sources"
	tmp_dir_stock := "tmp_grafter_tests/dest"
	//abs_tmp_dir_stock := filepath.Abs(tmp_dir_stock)
	tmp_dir_home := "tmp_grafter_tests/home"
	abs_tmp_dir, err := filepath.Abs(tmp_dir_src)
	if err != nil {
		t.Errorf("Error turning path [%v] absolute\n", tmp_dir_src)
	}
	var a_hash_original hash.Hash
	var a_hash_grafted hash.Hash
	var b_hash_original hash.Hash
	var b_hash_grafted hash.Hash
	a_test_file_path := filepath.Join(tmp_dir_src, "a")
	a_test_file_path_abs := filepath.Join(abs_tmp_dir, "a")
	b_test_file_path := filepath.Join(tmp_dir_src, "b")
	b_test_file_path_abs := filepath.Join(abs_tmp_dir, "b")
	files_test_slice := []string{a_test_file_path, b_test_file_path}
	// NOTE: Create directory where test files will live
	err = os.MkdirAll(tmp_dir_src, 0o755)
	if err != nil {
		t.Error(err)
	}
	// NOTE: Create the file a for testing
	a_test_file, err := os.OpenFile(a_test_file_path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Errorf("Error opening file [%v]: %v\n", a_test_file_path, err)
	}
	// NOTE: Write to file a
	_, err = a_test_file.WriteString(test_string)
	if err != nil {
		t.Errorf("Error writing to file [%v]: %v\n", a_test_file_path, err)
	}
	// NOTE: Close file a and reopen read only
	a_test_file.Close()
	a_test_file, err = os.Open(a_test_file_path)
	if err != nil {
		t.Errorf("Error opening file in read only [%v]: %v\n", a_test_file_path, err)
	}
	// NOTE: Hash the file a
	a_hash_original = sha256.New()
	_, err = io.Copy(a_hash_original, a_test_file)
	if err != nil {
		t.Errorf("Error hashing a: %v\n", err)
	}
	// NOTE: Close file a again
	a_test_file.Close()

	// NOTE: Create the file b for testing
	b_test_file, err := os.OpenFile(b_test_file_path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Errorf("Error opening file [%v]: %v\n", b_test_file_path, err)
	}
	// NOTE: Write to file b
	_, err = b_test_file.WriteString(test_string2)
	if err != nil {
		t.Errorf("Error writing to file [%v]: %v\n", b_test_file_path, err)
	}

	// NOTE: Close file b and reopen read only
	b_test_file.Close()
	b_test_file, err = os.Open(b_test_file_path)
	if err != nil {
		t.Errorf("Error opening file in read only [%v]: %v\n", a_test_file_path, err)
	}
	// NOTE: Hash the file b
	b_hash_original = sha256.New()
	_, err = io.Copy(b_hash_original, b_test_file)
	if err != nil {
		t.Errorf("Error hashing b: %v\n", err)
	}
	// NOTE: Close file b again
	b_test_file.Close()

	var runtime_scions_test []scion = make([]scion, 0, len(files_test_slice))
	err = scionsFromArgs(&runtime_scions_test, files_test_slice, tmp_dir_stock, tmp_dir_home, true, true)
	if err != nil {
		t.Errorf("Error building scions: %v\n", err)
	}

	// NOTE: Test a
	fmt.Println()
	fmt.Println("Asserting scion [a] against runtime_scions")
	a_dest := filepath.Join(tmp_dir_stock, "a")
	a_sci_manual := scion{
		ID:           "a",
		Stem:         "a",
		Path:         a_dest,
		OriginalPath: a_test_file_path_abs,
		Dir:          false,
		Perms:        0o644,
		Note:         "",
		Exists:       true}
	scion_assert(runtime_scions_test[0], a_sci_manual)
	// NOTE: Test b
	fmt.Println()
	fmt.Println("Asserting scion [a] against runtime_scions")
	scion_assert(runtime_scions_test[0], a_sci_manual)
	b_dest := filepath.Join(tmp_dir_stock, "a")
	b_sci_manual := scion{
		ID:           "b",
		Stem:         "b",
		Path:         b_dest,
		OriginalPath: b_test_file_path_abs,
		Dir:          false,
		Perms:        0o644,
		Note:         "",
		Exists:       true}
	scion_assert(runtime_scions_test[1], b_sci_manual)

	// TODO: Graft sciuons and hash to assert
	var grafts_test graftage
	grafts_test.Init(tmp_dir, tmp_dir_stock)
	grafts_test.GraftScions(runtime_scions_test, true)
	graft_assert(grafts_test, runtime_scions_test[0], runtime_scions_test[0].ID, tmp_dir_stock)
	graft_assert(grafts_test, runtime_scions_test[1], runtime_scions_test[1].ID, tmp_dir_stock)

	a_graft_file, err := os.Open(grafts_test.Scions["a"].Path)
	if err != nil {
		t.Errorf("Error opening the grafted file [%v]: %v\n", grafts_test.Scions["a"].Path, err)
	}
	defer a_graft_file.Close()
	a_hash_grafted = sha256.New()
	_, err = io.Copy(a_hash_grafted, a_graft_file)
	if err != nil {
		t.Errorf("Error hashing [%v]: %v\n", grafts_test.Scions["a"].Path, err)
	}
	a_hash_original_str := hex.EncodeToString(a_hash_original.Sum(nil))
	a_hash_grafted_str := hex.EncodeToString(a_hash_grafted.Sum(nil))
	dumb_assert(
		a_hash_original_str,
		a_hash_grafted_str,
		"\n[a] Asserting original hash against grafted hash\n")
	b_graft_file, err := os.Open(grafts_test.Scions["b"].Path)
	if err != nil {
		t.Errorf("Error opening the grafted file [%v]: %v\n", grafts_test.Scions["b"].Path, err)
	}
	b_hash_grafted = sha256.New()
	_, err = io.Copy(b_hash_grafted, b_graft_file)
	if err != nil {
		t.Errorf("Error hashing [%v]: %v\n", grafts_test.Scions["s"].Path, err)
	}
	b_hash_original_str := hex.EncodeToString(b_hash_original.Sum(nil))
	b_hash_grafted_str := hex.EncodeToString(b_hash_grafted.Sum(nil))
	dumb_assert(
		b_hash_original_str,
		b_hash_grafted_str,
		"[b] Asserting original hash against grafted hash\n")
	//os.RemoveAll(tmp_dir_src)
}
func graft_assert(g graftage, rt_scion scion, id string, stock string) {
	// NOTE: Stock
	dumb_assert(g.StockDir, stock, "")
	s := g.Scions[id]
	fmt.Println()
	fmt.Printf("Asserting ID [%s] against graft scion\n", id)
	scion_assert(s, rt_scion)
}
func scion_assert(s, t scion) {
	// NOTE: ID
	dumb_assert(s.ID, t.ID, "ID\n")
	// NOTE: Stem
	dumb_assert(s.Stem, t.Stem, "Stem\n")
	// NOTE: Path
	dumb_assert(s.Path, t.Path, "Path\n")
	// NOTE: ORiginalPath
	dumb_assert(s.OriginalPath, t.OriginalPath, "OriginalPath\n")
	// NOTE: Dir
	dumb_assert(s.Dir, t.Dir, "Dir\n")
	// NOTE: Size?
	// NOTE: Perms
	dumb_assert(s.Perms, t.Perms, "Perms\n")
	// NOTE: Note
	dumb_assert(s.Note, t.Note, "Note\n")
	// NOTE: Exists
	dumb_assert(s.Exists, t.Exists, "Exists\n")
}

func TestDirectories(t *testing.T) {}
