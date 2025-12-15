package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand/v2"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// NOTE: We implement this type so we can
// implkement a custom writer for it
// to take advantage of TeeReader
type transferStatus struct {
	bytesWritten uint64
}

// NOTE: teeReader sends to the writer, everything that it reads
// This allows us to put a file we want to copy on the reader
// and make a struct that satisfies the writer interface
// allowing us to read n amount of bytes from the source file
// and then send those same bytes to a writer
// which in this case its this function
// AAAAAAAAAAAAAAAAAAAND we can keep track of everything
// We should implement a way to do it every n amount of bytes,
// I say every GB
func (tf *transferStatus) Write(data []byte) (int, error) {
	tf.bytesWritten += uint64(len(data))
	// Jump to the beginning of the previous line (assumes we println)
	fmt.Printf("\x1b[1F")
	// Clear the line
	fmt.Printf("\x1b[2K")
	// FIXME
	fmt.Printf("cool i got %f\n", float64(tf.bytesWritten)/1000000000)
	return len(data), nil
}

// NOTE: Would be cool to have an error datatype for grafter
// also maybe change the name from grafter?

// This is the main data type for the individual things we graft
// We should try to not lose info, we could
// save the whole fs.FileInfo but I will refrain for now
// MORSEL: A scion is what we insert into the root
type scion struct {
	// ID, will be similar to stem
	// Based on filename but modified if necessary
	ID string
	// File name
	// We can obtain this from stat
	Stem string
	// Path where it is currently stored
	Path string
	// Path where it was originally stored
	OriginalPath string
	// If its a file or nah
	Dir bool
	// Size of file
	Size int64
	// Mode bits
	Perms fs.FileMode
	// Date it was grafted
	Date time.Time
	// Additional notes
	Note string
	// TODO: Hash
	// TODO: Inputs that generated this file
	// TODO: Workflow used
}

type graftage struct {
	// This is where the scion will go
	// NOTE: This will be an option later on
	// MORSEL: This is the stock or rootstock of the graft, where the scion is placed
	StockDir string
	// This is all the files/paths we will graft
	Scions []scion
}

// This function takes in a file name and verifies there's no scion
// in the graft that has a similar name
// if we find it we either update it or delete it
// THINK: Perhaps returns a pointer?
func (g *graftage) Find(n string, u bool) error { return nil }
func (g *graftage) WriteConfig(config string) error {
	g_b, err := json.MarshalIndent(g, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile(config, g_b, 0o644)
	if err != nil {
		return err
	}
	return nil
}

// FIXME: I don't like having the config file here...
// FIXME: Change to use ID and not stem
func (g *graftage) Remove(list []string, config string) error {
	// if we found anything at all
	found := false
list_loop:
	for _, v := range list {
		for j, x := range g.Scions {
			if v == x.ID {
				// We put the last element wherever
				// the element we want to delete is at
				// And then overwrite the array
				g.Scions[j] = g.Scions[len(g.Scions)-1]
				g.Scions[len(g.Scions)-1] = scion{}
				g.Scions = g.Scions[:len(g.Scions)-1]
				log.Printf("Removing [%s] with ID [%s] from the file system...", x.Path, x.ID)
				err := os.RemoveAll(x.Path)
				if err != nil {
					return err
				}
				found = true
				continue list_loop
			}
		}
		if found == false {
			log.Printf("Could not find ID [%s]", v)
		}
	}
	if found {

		log.Println("Saving changes to config file...")
		err := g.WriteConfig(config)
		if err != nil {
			return err
		}
		log.Println("Success!")
	}
	return nil
}

// This function takes in a scion and updates the values of the member scion
func (s *scion) Update(s2 scion) error { return nil }

func readDirNCreate() error { return nil }

func CopyFile(dst string, src string, perms fs.FileMode) error {
	// Open file we will read
	srcFi, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFi.Close()
	// Open file where we will write
	dstFi, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, perms)
	if err != nil {
		return err
	}
	defer dstFi.Close()
	io.Copy(dstFi, srcFi)
	// Ensure the permission is respected
	// since sometimes the permission is not
	// added at creation
	os.Chmod(dst, perms)
	return nil
}

// p is the directory where all these entries are in
// FIXME: Need to rename variables in this function
func DigDirNCreate(d []fs.DirEntry, dstPath string, srcPath string) error {
	for _, f := range d {
		innerFiPath := filepath.Join(srcPath, f.Name())
		innerFiStat, err := os.Stat(innerFiPath)
		if err != nil {
			log.Fatal(err)
		}
		dstFiPath := filepath.Join(dstPath, f.Name())
		if f.IsDir() {
			// Create the dst dir so when its time to copy we can finally do it
			err = os.MkdirAll(dstFiPath, innerFiStat.Mode())
			if err != nil {
				return err
			}
			newSrcDirEnt, err := os.ReadDir(innerFiPath)
			if err != nil {
				return err
			}
			err = DigDirNCreate(newSrcDirEnt, dstFiPath, innerFiPath)
			if err != nil {
				return nil
			}
		} else {
			log.Printf("Copying [%s] to [%s]\n", innerFiPath, dstFiPath)
			err := CopyFile(dstFiPath, innerFiPath, innerFiStat.Mode())
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return nil
}

// Also FIXME: Add a check of file name
// TODO: Add checking if the config file exists first and if not create with defaults
// TODO: add an option to make the
// Config is the path where we will load the user config from
func (g *graftage) Init(config string, stock string) error {

	grafted_files_bytes, err := os.ReadFile(config)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("No config file, creating...")
		g.StockDir = stock
	} else if err != nil {
		log.Fatal(err)
	} else {
		err = json.Unmarshal(grafted_files_bytes, &g)
		if err != nil {
			log.Fatal(err)
		}
		if g.StockDir == "" {
			log.Println("Empty graft location, adding default...")
			g.StockDir = stock
		}
	}
	if !filepath.IsAbs(g.StockDir) {
		g.StockDir, err = filepath.Abs(g.StockDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func scionsFromArgs(rt_scions *[]scion, args []string, stock string, user_home string) error {
	for _, v := range args {
		var scion_temp = scion{}
		// XXX: Should we stat first or absolute path first?
		// FIXME: In the future we can add recovering from failed abs
		// and failed stat
		// grafted path
		var scion_path string
		var err error
		if !filepath.IsAbs(v) {
			// We know its not absolute
			// And we only want to handle in a special manner
			// tilde, all other forms like ./
			// we can leave to filepath.Abs()
			if v == "~" {
				scion_path = user_home
			} else if strings.HasPrefix(v, "~/") {
				scion_path = filepath.Join(user_home, v[2:])
			} else {
				scion_path, err = filepath.Abs(v)
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			scion_path = v
		}
		// SECTION: stat path
		stat_result, err := os.Stat(scion_path)
		if err != nil {
			log.Fatal(err)
		}
		// TODO:ID
		scion_temp.ID = stat_result.Name()
		// TODO: stem
		scion_temp.Stem = stat_result.Name()
		// TODO: path
		// We want to join this
		// FIXME: Add a runtime option to use a different directory
		scion_temp.Path = filepath.Join(stock, scion_temp.Stem)
		// TODO: original path
		scion_temp.OriginalPath = scion_path
		// SECTION: Check if its a dir or a file
		if stat_result.IsDir() {
			scion_temp.Dir = true
		}
		// TODO: Size
		scion_temp.Size = stat_result.Size()
		// TODO: perms
		scion_temp.Perms = stat_result.Mode()
		// TODO: date
		scion_temp.Date = time.Now()
		// TODO: note?
		scion_temp.Note = ""
		// TODO: Hash
		// TODO: Inputs that generated this file
		// TODO: Workflow used

		// Finally add to scions to graft later on
		// FIXME: This needs to be a runtime scion and not the same scion we unmarshal
		// so we can properly verify against our graft config
		*rt_scions = append(*rt_scions, scion_temp)
	}
	return nil
}

// Grafts everything in the slice of scions
func (g *graftage) GraftScions(rt_scions []scion) error {
	// For evaluates the variable we will range over ONCE
	// so we are free to modify the variable and append to it
	// without worrying the loop will extend once again
	// https://go.dev/ref/spec#For_statements
	for _, x := range rt_scions {
		log.Printf("Grafting [%s] to [%s]\n", x.OriginalPath, x.Path)
		// Maybe we could move this to scionsFromArgs
		for _, v := range g.Scions {
			// Ensure the ID is the same and the original path is the same
			// although we can probably just use the original path
			if v.ID == x.ID && v.OriginalPath == x.OriginalPath {
				log.Printf("Found [%v] in the user graft...\n",
					x.ID)

				// Generate a random 4 digit number to append at the end
				r := rand.Uint32() >> 19
				digit := uint64(r)
				digit_s := strconv.FormatUint(digit, 10)
				x.ID = x.ID + "-" + digit_s
				if x.Dir {
					// Since its a directory we only need to append
					x.Path = x.Path + "-" + digit_s
				} else {
					// we need to extract the directory and recreate
					// the full file path
					scion_dir := filepath.Dir(x.Path)
					x.Path = filepath.Join(scion_dir, x.ID)
				}
				log.Printf("Creating copy with ID [%v]", x.ID)
				// Append at the end of the string
				//rt_scions[j].ID = new_scion_id_t
				// The scion is already in the main graft
				break
			}
		}
		if x.Dir {
			files, err := os.ReadDir(x.OriginalPath)
			if err != nil {
				log.Fatal(err)
			}
			// FIXME / DOCS : in the docs we need to mention that
			// he directories inherit the permissions from the main grafted dir
			err = os.MkdirAll(x.Path, x.Perms)
			if err != nil {
				log.Fatal(err)
			}
			err = DigDirNCreate(files, x.Path, x.OriginalPath)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// First we create the directories
			dir_only := filepath.Dir(x.Path)
			err := os.MkdirAll(dir_only, 0o755)
			if err != nil {
				log.Fatal(err)
			}
			err = CopyFile(x.Path, x.OriginalPath, x.Perms)
			if err != nil {
				log.Fatal(err)
			}
		}
		// Finally add the new member to the graft
		// The range grabs a copy of the variable its ranging over,
		// so it is safe to modify the original while looping
		g.Scions = append(g.Scions, x)
	}
	return nil
}

// The way this should works is that we go through the listed files
// and directories with an array of some sort
func main() {
	if len(os.Args) < 2 {
		fmt.Println("USAGE: grafter [file1] [file2] ... [fileN]")
		fmt.Println("grafter -l will list all of your grafted paths")
		fmt.Println("grafter -r [ID1] [ID2] ... [IDN] will remove the files from the graft")
		os.Exit(1)
	}
	// For tilde expansion
	runtime_user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	rt_user_home_dir := runtime_user.HomeDir

	// So there's a future problem here
	// What happens when the list is so long we
	// need to traverse it to check if a file is already there

	// CAUTION: This needs to be considered when we move to the new file handling
	// empty but has a capacity that is non-zero
	// TODO: Add a function for validating a scion is not in the list
	// TODO: Add a function for updating a scion
	var grafts graftage
	config_file := filepath.Join(rt_user_home_dir, "grafts.json")
	stock_dir := filepath.Join(rt_user_home_dir, "Grafts")
	grafts.Init(config_file, stock_dir)

	// FIXME: Maybe just list the file names or whatever identifier we are using
	if os.Args[1] == "-l" {
		for _, v := range grafts.Scions {
			fmt.Printf("Date: %-12s | ID: %-12s | Original: %s\n", v.Date.Format(time.RFC822), v.ID, v.OriginalPath)
		}
		os.Exit(0)
	} else if os.Args[1] == "-r" && len(os.Args) > 2 {
		err := grafts.Remove(os.Args[2:], config_file)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	var runtime_scions []scion = make([]scion, 0, len(os.Args)-1)
	// This is where we would add a condition to indicate if in THIS run
	// of the program we would use a different stock directory than
	// the one in the config file/default one
	err = scionsFromArgs(&runtime_scions, os.Args[1:], grafts.StockDir, rt_user_home_dir)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Add a section validating that the runtime scions are different from the ones in the graftr
	// if we find one thats similar we either update or remove it
	// this can be an option too
	// NOTE: We only need to do operations on the runtime scions and at the end update the
	// json...
	// XXX: We will validate the runtime in its own section so we can give the user the option to
	// remove it of the list or update it, although for now we will just ignore the runtime one
	// We will loop the existing grafts and compare against the runtime scions,
	// as opposed to loop the runtime scions and compare against the
	// existing scions
	// I think this is indicative that we need a better data structure than
	// an array with all the scions loll
	// We could in theory add the looping of the runtime scions here but no,
	// not for now
	err = grafts.GraftScions(runtime_scions)
	if err != nil {
		log.Fatal(err)
	}
	err = grafts.WriteConfig(config_file)
	if err != nil {
		log.Println("Error writing to config file...")
		log.Fatal(nil)
	}
	// TODO / FIXME So we need a way to save to a file,
	// Need to create a new syntax for that
	// Probably parsing since I dont want to use json
	// We need to think about the date format
	// solution probably related to the many ways golang gives to represent dates
}
