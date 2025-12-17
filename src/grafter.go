package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand/v2"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const GraftVersion = "0003"

// NOTE: As of 17/12/2025, it seems that WriteConfig does not turn path into absolutes

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
	// TODO: Command that generated this workflow
	// TODO: Workflow used
	// If it has been grafted yet
	Exists bool
}

type graftage struct {
	// This is where the scion will go
	// NOTE: This will be an option later on
	// MORSEL: This is the stock or rootstock of the graft, where the scion is placed
	StockDir string
	// This is all the files/paths we will graft
	Scions map[string]scion
}

// Also FIXME: Add a check of file name
// TODO: Add checking if the config file exists first and if not create with defaults
// TODO: add an option to make the
// Config is the path where we will load the user config from
// Init function opens the config file and checks if theres anything there
func (g *graftage) Init(config string, stock string) error {
	// Need to init this before anything can be put inside
	g.Scions = make(map[string]scion)
	grafted_files_bytes, err := os.ReadFile(config)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("No config file, using defaults...")
		// Retrieve the value that will be used inside the config
		g.StockDir = stock
	} else if err != nil {
		return err
	} else {
		err = json.Unmarshal(grafted_files_bytes, &g)
		if err != nil {
			return err
		}
		if g.StockDir == "" {
			log.Println("Empty graft location, adding default...")
			g.StockDir = stock
		}
	}
	if !filepath.IsAbs(g.StockDir) {
		g.StockDir, err = filepath.Abs(g.StockDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// Writes the graft to the config file
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

// Grafts everything in the slice of scions
func (g *graftage) GraftScions(rt_scions []scion, move bool) error {
	// For evaluates the variable we will range over ONCE
	// so we are free to modify the variable and append to it
	// without worrying the loop will extend once again
	// https://go.dev/ref/spec#For_statements
	for i, x := range rt_scions {
		log.Printf("Grafting [%s]\n", x.OriginalPath)
		// Maybe we could move this to scionsFromArgs
		s := g.Scions[x.ID]
		if s != (scion{}) {
			// Ensure the ID is the same and the original path is the same
			// although we can probably just use the original path
			if s.ID == x.ID && s.OriginalPath == x.OriginalPath {
				log.Printf("Found [%v] in the user graft...\n",
					x.ID)

				// Generate a random 4 digit number to append at the end
				r := rand.Uint32() >> 19
				digit := uint64(r)
				digit_s := strconv.FormatUint(digit, 10)
				x.ID = x.ID + "-" + digit_s
				if move {
					if x.Dir {
						// Since its a directory we only need to append
						// Add option to only fetch the indicated ones
						x.Path = x.Path + "-" + digit_s
					} else {
						// we need to extract the directory and recreate
						// the full file path
						scion_dir := filepath.Dir(x.Path)
						x.Path = filepath.Join(scion_dir, x.ID)
					}
				}
				log.Printf("Creating copy with ID [%v]", x.ID)
				// Append at the end of the string
				//rt_scions[j].ID = new_scion_id_t
				// The scion is already in the main graft
			}
		}
		// We wont move it
		// Since rt_scions has not been modified
		if !x.Exists || !move || (move && x.OriginalPath == rt_scions[i].Path) {
			g.Scions[x.ID] = x
			continue
		}
		err := x.WriteScion()
		if err != nil {
			return nil
		}
		// Finally add the new member to the graft
		// The range grabs a copy of the variable its ranging over,
		// so it is safe to modify the original while looping
		g.Scions[x.ID] = x
	}
	return nil
}

// FIXME: I don't like having the config file here...
// FIXME: Change to use ID and not stem
func (g *graftage) Remove(list []string, config string) error {
	// if we found anything at all
	found := false
	for _, v := range list {
		s := g.Scions[v]
		if s != (scion{}) {
			log.Printf("Removing [%s] with ID [%s] from the file system...", s.Path, s.ID)
			delete(g.Scions, v)
			err := os.RemoveAll(s.Path)
			if err != nil {
				return err
			}
			found = true
			continue
		}
		log.Printf("Could not find ID [%s]", v)
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

// Fetches all the scions that have not been grafted yet
func (g *graftage) FetchMissing(config string) error {
	// Add option to only fetch the indicated ones
	for _, v := range g.Scions {
		if !v.Exists {
			// SECTION: Add missing things in graft
			stat_result, err := os.Stat(v.OriginalPath)
			if err != nil {
				log.Fatal(err)
			}
			// TODO:ID
			v.ID = stat_result.Name()
			// TODO: stem
			v.Stem = stat_result.Name()
			// SECTION: Check if its a dir or a file
			if stat_result.IsDir() {
				v.Dir = true
			}
			// TODO: Size
			v.Size = stat_result.Size()
			// TODO: perms
			v.Perms = stat_result.Mode().Perm()

			v.WriteScion()
			g.Scions[v.ID] = v
		}
	}
	err := g.WriteConfig(config)
	if err != nil {
		log.Println("Error writing to config file...")
		log.Fatal(err)
	}
	return nil
}

// Copies the listed ID inside dstPath
func (g *graftage) CopyScions(ids []string, dstPath string) error {
	err := os.MkdirAll(dstPath, 0o755)
	if err != nil {
		return err
	}
	for _, x := range ids {
		s := g.Scions[x]
		if s != (scion{}) {
			dstPathComplete := filepath.Join(dstPath, s.ID)
			if s.Dir {
				files, err := os.ReadDir(s.Path)
				if err != nil {
					return err
				}
				// NOTE: Need to have this because we are placing it
				// inside the directory we pass, so we need to join it
				err = os.MkdirAll(dstPathComplete, s.Perms)
				if err != nil {
					return err
				}
				_, err = DigDirNCreate(files, dstPathComplete, s.Path)
				if err != nil {
					return err
				}
			} else {
				err = CopyFile(dstPathComplete, s.Path, s.Perms)
			}
		}
	}
	return nil
}

// FIXME: Could perhaps make it take a string as an arg to allow for flipping
// original with a path
// Writes the scion to the filesystem aka add it to the graft folder
// FIXME: Change name
func (s *scion) WriteScion() error {
	if s.Dir {
		files, err := os.ReadDir(s.OriginalPath)
		if err != nil {
			return err
		}
		// FIXME / DOCS : in the docs we need to mention that
		// he directories inherit the permissions from the main grafted dir
		err = os.MkdirAll(s.Path, s.Perms)
		if err != nil {
			return err
		}
		s.Size, err = DigDirNCreate(files, s.Path, s.OriginalPath)
		if err != nil {
			return err
		}
	} else {
		// First we create the directories
		dir_only := filepath.Dir(s.Path)
		err := os.MkdirAll(dir_only, 0o755)
		if err != nil {
			return err
		}
		err = CopyFile(s.Path, s.OriginalPath, s.Perms)
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyFile(dst string, src string, perms fs.FileMode) error {
	log.Printf("Copying [%s] to [%s]\n", src, dst)
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
func DigDirNCreate(d []fs.DirEntry, dstPath string, srcPath string) (int64, error) {
	var size_total int64
	for _, f := range d {
		innerFiPath := filepath.Join(srcPath, f.Name())
		innerFiStat, err := os.Stat(innerFiPath)
		if err != nil {
			log.Fatal(err)
		}
		dstFiPath := filepath.Join(dstPath, f.Name())
		if f.IsDir() {
			// Create the dst dir so when its time to copy we can finally do it
			err = os.MkdirAll(dstFiPath, innerFiStat.Mode().Perm())
			if err != nil {
				return 0, err
			}
			newSrcDirEnt, err := os.ReadDir(innerFiPath)
			if err != nil {
				return 0, err
			}
			siz, err := DigDirNCreate(newSrcDirEnt, dstFiPath, innerFiPath)
			if err != nil {
				return 0, nil
			}
			size_total += siz
		} else {
			err := CopyFile(dstFiPath, innerFiPath, innerFiStat.Mode().Perm())
			if err != nil {
				log.Fatal(err)
			}
			size_total += innerFiStat.Size()
		}
	}
	return size_total, nil
}

// temp_graft_dir is the directory where these
func scionsFromArgs(rt_scions *[]scion, args []string, stock string, user_home string, exist bool, move bool) error {
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

		// NOTE: Shared values
		// TODO: original path
		scion_temp.OriginalPath = scion_path
		// TODO: note?
		scion_temp.Note = ""
		// TODO: date
		scion_temp.Date = time.Now()

		if !exist {
			s_name := filepath.Base(v)
			scion_temp.ID = s_name
			scion_temp.Stem = s_name
			scion_temp.Path = filepath.Join(stock, scion_temp.Stem)
			scion_temp.Exists = false
		} else {
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
			if move {
				scion_temp.Path = filepath.Join(stock, scion_temp.Stem)
				fmt.Println(scion_temp.Path)
			} else {
				scion_temp.Path = scion_path
			}
			// SECTION: Check if its a dir or a file
			if stat_result.IsDir() {
				scion_temp.Dir = true
			}
			// TODO: Size
			scion_temp.Size = stat_result.Size()
			// TODO: perms
			scion_temp.Perms = stat_result.Mode().Perm()
			// TODO: Grafted
			// If it wont be grafted we pass a false
			scion_temp.Exists = true
		}
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

type rt_options struct {
	version   bool
	list      bool
	fetch     bool
	remove    bool
	exist     bool
	move      bool
	copy      bool
	copyDir   string
	graft_dir string
}

// The way this should works is that we go through the listed files
// and directories with an array of some sort
func main() {
	stdin_arr := make([]string, 0)
	in_stat, _ := os.Stdin.Stat()
	// Equivalent to !isatty()
	if in_stat.Mode()&os.ModeCharDevice == 0 {
		in := bufio.NewScanner(os.Stdin)
		in.Split(bufio.ScanWords)
		for in.Scan() {
			stdin_arr = append(stdin_arr, in.Text())
		}
	}
	// SECTION: Init of runtime stuff
	var opts rt_options
	flag.BoolVar(&opts.version, "version", false, "Show the current version of grafter.")
	flag.BoolVar(&opts.list, "list", false, "List everything in the user graft.")
	flag.BoolVar(&opts.fetch, "fetch", false, "Validate and fetch missing files in the user graft.")
	flag.BoolVar(&opts.remove, "remove", false, "Remove the listed IDs from the user graft.")
	flag.BoolVar(&opts.exist, "exist", true, "If -exist=0 the files will only be grafted to the list.")
	// This flag defaults to true, I think that in general, with the use cases im thinking
	// you will want to make copies of the file
	flag.BoolVar(&opts.move, "move", true, "If the file should be moved to the graft directory.")
	flag.BoolVar(&opts.copy, "copy", false, "Copy the files of the listed IDs to destdir.")
	flag.StringVar(&opts.copyDir, "destdir", "", "Directory to copy the selected files to.")
	flag.StringVar(&opts.graft_dir, "graftdir", "", "Directory to graft the listed files to")
	flag.Parse()
	var arg_list []string
	if len(stdin_arr) == 0 {
		arg_list = flag.Args()
	} else {
		arg_list = stdin_arr
	}
	// rt_files_exist := true
	// if opts.exist {
	// 	rt_files_exist = false
	// }
	// For tilde expansion
	runtime_user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	rt_user_home_dir := runtime_user.HomeDir
	runtime_stock_dir := opts.graft_dir
	if runtime_stock_dir == "" {
		runtime_stock_dir = filepath.Join(rt_user_home_dir, "Grafts")
	} else if !filepath.IsAbs(runtime_stock_dir) {
		runtime_stock_dir, err = filepath.Abs(runtime_stock_dir)
		if err != nil {
			log.Fatal(err)
		}
	}

	// So there's a future problem here
	// What happens when the list is so long we
	// need to traverse it to check if a file is already there

	// CAUTION: This needs to be considered when we move to the new file handling
	// empty but has a capacity that is non-zero
	// TODO: Add a function for validating a scion is not in the list
	// TODO: Add a function for updating a scion
	var grafts graftage
	config_file := filepath.Join(rt_user_home_dir, "grafts.json")
	err = grafts.Init(config_file, runtime_stock_dir)
	if err != nil {
		log.Fatal(err)

	}

	if opts.version {
		fmt.Println("Grafter version:", GraftVersion)
		b, _ := debug.ReadBuildInfo()
		fmt.Println(b.String())
		os.Exit(0)
	} else if opts.list {
		for _, v := range grafts.Scions {
			fmt.Printf("Date: %-12s | ", v.Date.Format(time.RFC822))
			fmt.Printf("ID: %-12s | ", v.ID)
			fmt.Printf("Original: %s\n", v.OriginalPath)
		}
		os.Exit(0)
	} else if opts.fetch {
		grafts.FetchMissing(config_file)
		os.Exit(0)
	} else if opts.copy {
		if opts.copyDir == "" {
			log.Println("Empty destdir flag.")
			flag.PrintDefaults()
		}
		err := grafts.CopyScions(arg_list, opts.copyDir)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	} else if opts.remove {
		err := grafts.Remove(arg_list, config_file)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	// SECTION: actual program processing
	var runtime_scions []scion = make([]scion, 0, len(arg_list))
	// This is where we would add a condition to indicate if in THIS run
	// of the program we would use a different stock directory than
	// the one in the config file/default one

	err = scionsFromArgs(&runtime_scions, arg_list, runtime_stock_dir, rt_user_home_dir, opts.exist, opts.move)
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
	err = grafts.GraftScions(runtime_scions, opts.move)
	if err != nil {
		log.Fatal(err)
	}
	err = grafts.WriteConfig(config_file)
	if err != nil {
		log.Println("Error writing to config file...")
		log.Fatal(err)
	}
	// TODO / FIXME So we need a way to save to a file,
	// Need to create a new syntax for that
	// Probably parsing since I dont want to use json
	// We need to think about the date format
	// solution probably related to the many ways golang gives to represent dates
}
