package main

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"testing"
)

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
	tmp_dir := "tmp_grafter_tests/sources"
	tmp_dir_dest := "tmp_grafter_tests/dest"
	tmp_dir_home := "tmp_grafter_tests/home"
	abs_tmp_dir, err := filepath.Abs(tmp_dir)
	if err != nil {
		t.Errorf("Error turning path [%v] absolute\n", tmp_dir)
	}
	var a_hash_original hash.Hash
	//var a_hash_grafted hash.Hash
	var b_hash_original hash.Hash
	a_test_file_path := filepath.Join(tmp_dir, "a")
	a_test_file_path_abs := filepath.Join(abs_tmp_dir, "a")
	fmt.Println(a_test_file_path_abs)
	b_test_file_path := filepath.Join(tmp_dir, "b")
	files_test_slice := []string{a_test_file_path, b_test_file_path}
	// NOTE: Create directory where test files will live
	err = os.MkdirAll(tmp_dir, 0o755)
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
		t.Errorf("Error hashing: %v\n", err)
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
		t.Errorf("Error hashing: %v\n", err)
	}
	// NOTE: Close file b again
	b_test_file.Close()

	fmt.Printf("Original hash [a]: %x\n", a_hash_original.Sum(nil))
	fmt.Printf("Original hash [b]: %x\n", b_hash_original.Sum(nil))

	var runtime_scions_test []scion = make([]scion, 0, len(files_test_slice))
	err = scionsFromArgs(&runtime_scions_test, files_test_slice, tmp_dir_dest, tmp_dir_home, true)
	if err != nil {
		t.Errorf("Error building scions: %v\n", err)
	}
	// Assertions
	// NOTE: ID
	dumb_assert(runtime_scions_test[0].ID, "a")
	// NOTE: Stem
	dumb_assert(runtime_scions_test[0].Stem, "a")
	// NOTE: Path
	a_dest := filepath.Join(tmp_dir_dest, "a")
	dumb_assert(runtime_scions_test[0].Path, a_dest)
	// NOTE: ORiginalPath
	dumb_assert(runtime_scions_test[0].OriginalPath, a_test_file_path_abs)
	// NOTE: Dir
	dumb_assert(runtime_scions_test[0].Dir, false)
	// NOTE: Size?
	// NOTE: Perms
	dumb_assert(runtime_scions_test[0].Perms, 0o644)
	// NOTE: Note
	dumb_assert(runtime_scions_test[0].Note, "")
	// NOTE: Grafted
	dumb_assert(runtime_scions_test[0].Grafted, true)

	// TODO: Graft sciuons and hash to assert
	os.RemoveAll(tmp_dir)
}

func dumb_assert[V comparable](a V, b V) error {
	if a != b {
		return fmt.Errorf("ASSERTION FAILED: %v != %v", a, b)
	} else {
		fmt.Printf("ASSERT: %v == %v\n", a, b)
		return nil
	}
}

func TestDirectories(t *testing.T) {}
